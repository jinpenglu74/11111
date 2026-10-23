package mai_onsyn.RewriteDubbingV2.velo;

import javafx.application.Platform;
import mai_onsyn.AnimeFX.Utls.AXTreeItem;
import mai_onsyn.VeloVoice.FrameFactory.MainFactory;
import mai_onsyn.VeloVoice.NetWork.TTS.ResumableTTSClient;
import mai_onsyn.VeloVoice.Text.TTS;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicReference;

import static mai_onsyn.VeloVoice.App.Runtime.CLIENT_TYPE;
import static mai_onsyn.VeloVoice.App.Runtime.disableNaturalTTS;
import static mai_onsyn.VeloVoice.App.Runtime.isTTSRunning;

/** Only writes final text and starts VeloVoice with the user's existing settings. */
public final class VeloVoiceAutomation {
    private VeloVoiceAutomation() {}
    public record Result(Path outputFolder, List<Path> files) {}

    public static Result dub(String baseName, String text) throws Exception {
        if (text == null || text.isBlank()) throw new IOException("最终配音文本为空");
        if (baseName == null || baseName.isBlank()) throw new IOException("任务名称为空");
        if (CLIENT_TYPE == ResumableTTSClient.ClientType.NATURAL && disableNaturalTTS) throw new IOException("Natural TTS 当前不可用，请先在 VeloVoice 中调整引擎");
        if (isTTSRunning.get()) throw new IOException("VeloVoice 当前已有任务正在运行");

        String safe = safeBase(baseName);
        Prepared prepared = prepareAndVerify(safe, text);
        Files.createDirectories(prepared.outputFolder());
        long started = System.currentTimeMillis();
        setRunning(true);
        try {
            TTS.start(prepared.item(), prepared.outputFolder().toFile());
            List<Path> generated = VeloVoiceOutputVerifier.collect(prepared.outputFolder(), safe, started);
            if (generated.isEmpty()) throw new IOException("VeloVoice 已结束，但没有发现本次 mp3/wav 产物：" + prepared.outputFolder());
            return new Result(prepared.outputFolder(), generated);
        } finally {
            setRunning(false);
        }
    }

    public static String configuredOutputFolder() throws Exception {
        return onFxThread(() -> MainFactory.getConfiguredAudioSaveFolder().toPath().toAbsolutePath().normalize()).toString();
    }

    private static Prepared prepareAndVerify(String base, String text) throws Exception {
        return onFxThread(() -> {
            AXTreeItem item = MainFactory.prepareAutomationText(base, text);
            String actual = MainFactory.getAutomationEditorText();
            if (item == null) throw new IOException("VeloVoice 未创建当前任务文本项");
            if (!sameText(actual, text)) throw new IOException("VeloVoice 文本写入后读回不一致，禁止开始配音");
            Path output = MainFactory.getConfiguredAudioSaveFolder().toPath().toAbsolutePath().normalize();
            return new Prepared(item, output);
        });
    }

    private static void setRunning(boolean value) throws Exception {
        onFxThread(() -> { isTTSRunning.set(value); return Boolean.TRUE; });
    }

    private static <T> T onFxThread(FxCall<T> call) throws Exception {
        if (Platform.isFxApplicationThread()) return call.run();
        CountDownLatch latch = new CountDownLatch(1);
        AtomicReference<T> result = new AtomicReference<>();
        AtomicReference<Throwable> error = new AtomicReference<>();
        Platform.runLater(() -> {
            try { result.set(call.run()); } catch (Throwable t) { error.set(t); } finally { latch.countDown(); }
        });
        if (!latch.await(15, TimeUnit.SECONDS)) throw new IOException("VeloVoice JavaFX 操作超时");
        if (error.get() != null) {
            if (error.get() instanceof Exception e) throw e;
            throw new IOException("VeloVoice JavaFX 操作失败", error.get());
        }
        return result.get();
    }

    static boolean sameText(String a, String b) { return normalize(a).equals(normalize(b)); }
    static String normalize(String s) { return s == null ? "" : s.replace("\r\n", "\n").replace('\r', '\n'); }
    static boolean isAudio(Path p) { return VeloVoiceOutputVerifier.isAudio(p); }
    static String safeBase(String name) {
        String base = name.toLowerCase().endsWith(".srt") ? name.substring(0, name.length() - 4) : name;
        return base.replaceAll("[\\\\/:*?\"<>|]", "_").trim();
    }
    private record Prepared(AXTreeItem item, Path outputFolder) {}
    @FunctionalInterface private interface FxCall<T> { T run() throws Exception; }
}
