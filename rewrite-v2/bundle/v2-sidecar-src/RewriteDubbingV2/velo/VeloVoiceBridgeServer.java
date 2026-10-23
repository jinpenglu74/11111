package mai_onsyn.RewriteDubbingV2.velo;

import com.alibaba.fastjson2.JSON;
import com.alibaba.fastjson2.JSONObject;
import javafx.application.Platform;
import javafx.stage.Stage;
import mai_onsyn.VeloVoice.App.Runtime;

import java.io.BufferedReader;
import java.io.BufferedWriter;
import java.io.InputStreamReader;
import java.io.OutputStreamWriter;
import java.net.InetAddress;
import java.net.ServerSocket;
import java.net.Socket;
import java.nio.charset.StandardCharsets;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Objects;
import java.util.concurrent.atomic.AtomicBoolean;

/** Loopback-only, nonce-authenticated newline JSON bridge. */
public final class VeloVoiceBridgeServer implements AutoCloseable {
    private final String nonce;
    private final Stage stage;
    private final ServerSocket server;
    private final AtomicBoolean running = new AtomicBoolean(true);
    private final Thread acceptThread;

    public VeloVoiceBridgeServer(String nonce, int port, Stage stage) throws Exception {
        if (nonce == null || nonce.length() < 24) throw new IllegalArgumentException("bridge nonce too short");
        this.nonce = nonce;
        this.stage = Objects.requireNonNull(stage);
        this.server = new ServerSocket(port, 16, InetAddress.getLoopbackAddress());
        this.acceptThread = Thread.ofPlatform().name("VeloVoiceBridgeAccept").daemon(true).unstarted(this::acceptLoop);
    }

    public int port() { return server.getLocalPort(); }
    public void start() { acceptThread.start(); }

    private void acceptLoop() {
        while (running.get()) {
            try {
                Socket socket = server.accept();
                Thread.ofVirtual().name("VeloVoiceBridgeClient").start(() -> handle(socket));
            } catch (Exception e) {
                if (running.get()) System.err.println("[VeloVoiceBridge] accept failed: " + e.getMessage());
            }
        }
    }

    private void handle(Socket socket) {
        try (socket;
             BufferedReader reader = new BufferedReader(new InputStreamReader(socket.getInputStream(), StandardCharsets.UTF_8));
             BufferedWriter writer = new BufferedWriter(new OutputStreamWriter(socket.getOutputStream(), StandardCharsets.UTF_8))) {
            socket.setSoTimeout(70 * 60 * 1000);
            String line = reader.readLine();
            Map<String, Object> response;
            try {
                JSONObject request = JSON.parseObject(line == null ? "{}" : line);
                response = dispatch(request);
            } catch (Throwable error) {
                response = error(error);
            }
            writer.write(JSON.toJSONString(response));
            writer.write("\n");
            writer.flush();
        } catch (Exception ignored) {}
    }

    Map<String, Object> dispatch(JSONObject request) throws Exception {
        if (!sameNonce(nonce, request.getString("token"))) throw new SecurityException("unauthorized");
        String action = request.getString("action");
        if (action == null) throw new IllegalArgumentException("missing action");
        return switch (action) {
            case "ping" -> ok(Map.of("service", "VeloVoice", "protocol", 1));
            case "status" -> ok(Map.of(
                    "running", Runtime.isTTSRunning.get(),
                    "clientType", String.valueOf(Runtime.CLIENT_TYPE),
                    "outputFolder", VeloVoiceAutomation.configuredOutputFolder()
            ));
            case "show" -> {
                Platform.runLater(() -> { stage.show(); stage.toFront(); stage.requestFocus(); });
                yield ok(Map.of("shown", true));
            }
            case "hide" -> {
                Platform.runLater(stage::hide);
                yield ok(Map.of("hidden", true));
            }
            case "dub" -> {
                String baseName = request.getString("baseName");
                String text = request.getString("text");
                VeloVoiceAutomation.Result result = VeloVoiceAutomation.dub(baseName, text);
                yield ok(Map.of(
                        "outputFolder", result.outputFolder().toString(),
                        "files", result.files().stream().map(path -> path.toAbsolutePath().normalize().toString()).toList()
                ));
            }
            case "shutdown" -> ok(Map.of("shutdown", true));
            default -> throw new IllegalArgumentException("unknown action: " + action);
        };
    }

    static Map<String, Object> ok(Map<String, ?> payload) {
        Map<String, Object> out = new LinkedHashMap<>();
        out.put("ok", true);
        out.putAll(payload);
        return out;
    }

    static Map<String, Object> error(Throwable error) {
        Map<String, Object> out = new LinkedHashMap<>();
        out.put("ok", false);
        out.put("error", error.getClass().getSimpleName());
        out.put("message", error.getMessage() == null ? "" : error.getMessage());
        return out;
    }

    static boolean sameNonce(String expected, String actual) {
        if (actual == null) return false;
        return java.security.MessageDigest.isEqual(
                expected.getBytes(StandardCharsets.UTF_8),
                actual.getBytes(StandardCharsets.UTF_8));
    }

    @Override public void close() throws Exception {
        running.set(false);
        server.close();
    }
}
