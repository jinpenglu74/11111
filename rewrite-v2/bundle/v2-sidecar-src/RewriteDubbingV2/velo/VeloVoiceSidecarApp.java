package mai_onsyn.RewriteDubbingV2.velo;

import javafx.application.Application;
import javafx.scene.Scene;
import javafx.scene.paint.Color;
import javafx.stage.Stage;
import mai_onsyn.AnimeFX.I18N;
import mai_onsyn.AnimeFX.layout.AutoPane;
import mai_onsyn.VeloVoice.App.AutoSaveListener;
import mai_onsyn.VeloVoice.App.ResourceManager;
import mai_onsyn.VeloVoice.App.WindowManager;
import mai_onsyn.VeloVoice.FrameFactory.FrameThemes;
import mai_onsyn.VeloVoice.FrameFactory.MainFactory;
import mai_onsyn.VeloVoice.NetWork.TTS.EdgeTTSClient;
import mai_onsyn.VeloVoice.NetWork.TTS.MultiTTSClient;
import mai_onsyn.VeloVoice.NetWork.TTS.NaturalTTSClient;
import mai_onsyn.VeloVoice.NetWork.TTS.ResumableTTSClient;
import org.apache.logging.log4j.Level;
import org.apache.logging.log4j.core.config.Configurator;

import java.util.List;
import java.util.Objects;

import static mai_onsyn.VeloVoice.App.Runtime.*;

/** Original VeloVoice UI + a private local bridge; no RewriteDubbing V1 task-center or AI code. */
public final class VeloVoiceSidecarApp extends Application {
    private VeloVoiceBridgeServer bridge;

    public static void main(String[] args) {
        initializeVeloVoice();
        Application.launch(VeloVoiceSidecarApp.class, args);
        AutoSaveListener.trySave();
    }

    @Override public void start(Stage stage) throws Exception {
        FrameThemes.setDarkMode(windowConfig.getBoolean("DarkMode"));
        String themeColor = windowConfig.getString("ThemeColor");
        try { FrameThemes.setThemeColor(Color.valueOf(themeColor)); } catch (Exception ignored) {}

        AutoPane root = new AutoPane();
        root.setOnMousePressed(ignored -> root.requestFocus());
        MainFactory.drawMainFrame(root);
        I18N.setLanguage(windowConfig.getString("Language"));
        themeManager.flushAll();
        Scene scene = new Scene(root, 1160, 760);
        stage.setMinWidth(960);
        stage.setMinHeight(640);
        stage.getIcons().add(ResourceManager.icon);
        stage.setTitle("VeloVoice 1.3.5 · RewriteDubbing V2");
        stage.setScene(scene);
        WindowManager.register(stage);
        WindowManager.initialize(stage);

        String token = System.getenv("REWRITE_VELOVOICE_TOKEN");
        if (token == null || token.isBlank()) token = System.getProperty("rewrite.velovoice.token", "");
        int port = parsePort(System.getenv("REWRITE_VELOVOICE_PORT"));
        bridge = new VeloVoiceBridgeServer(token, port, stage);
        bridge.start();
        System.out.println("VELOVOICE_BRIDGE_READY " + bridge.port());
        System.out.flush();

        stage.setOnCloseRequest(event -> {
            if (isTTSRunning.get()) {
                event.consume();
                stage.hide();
            }
        });
        stage.show();
    }

    @Override public void stop() throws Exception {
        if (bridge != null) bridge.close();
        AutoSaveListener.trySave();
    }

    private static int parsePort(String value) {
        try { int port = Integer.parseInt(value == null ? "0" : value); return port >= 0 && port <= 65535 ? port : 0; }
        catch (NumberFormatException ignored) { return 0; }
    }

    private static void initializeVeloVoice() {
        I18N.setLanguage(windowConfig.getString("Language"));

        EdgeTTSClient.setVoice(edgeTTSConfig.getString("SelectedModel"));
        EdgeTTSClient.setVoicePitch(edgeTTSConfig.getDouble("VoicePitch"));
        EdgeTTSClient.setVoiceRate(edgeTTSConfig.getDouble("VoiceRate"));
        EdgeTTSClient.setVoiceVolume(edgeTTSConfig.getDouble("VoiceVolume"));

        if (isDllLoadable("jacob-1.21-x64")) {
            List<String> voices = NaturalTTSClient.getVoiceList();
            if (voices.isEmpty()) disableNaturalTTS = true;
            else {
                String selected = naturalTTSConfig.getString("SelectedModel");
                if (!voices.contains(selected)) {
                    String first = voices.getFirst();
                    NaturalTTSClient.setVoice(first);
                    naturalTTSConfig.setString("SelectedModel", first);
                } else NaturalTTSClient.setVoice(selected);
            }
        } else disableNaturalTTS = true;
        NaturalTTSClient.setVoiceRate(naturalTTSConfig.getInteger("VoiceRate"));
        NaturalTTSClient.setVoiceVolume(naturalTTSConfig.getDouble("VoiceVolume"));

        MultiTTSClient.setVoiceModel(multiTTSConfig.getString("VoiceModel"));
        MultiTTSClient.setVoiceRate(multiTTSConfig.getInteger("VoiceRate"));
        MultiTTSClient.setVoiceVolume(multiTTSConfig.getInteger("VoiceVolume"));
        MultiTTSClient.setVoicePitch(multiTTSConfig.getInteger("VoicePitch"));

        for (ResumableTTSClient.ClientType type : ResumableTTSClient.ClientType.values()) {
            if (Objects.equals(type.getName(), config.getString("TTSEngine"))) CLIENT_TYPE = type;
        }
        Configurator.setRootLevel(Level.toLevel(config.getString("LogLevel"), Level.INFO));
    }

    private static boolean isDllLoadable(String dllName) {
        try { System.loadLibrary(dllName); return true; }
        catch (UnsatisfiedLinkError error) { return false; }
    }
}
