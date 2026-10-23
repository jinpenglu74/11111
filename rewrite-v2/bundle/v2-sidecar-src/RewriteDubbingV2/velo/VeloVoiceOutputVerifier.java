package mai_onsyn.RewriteDubbingV2.velo;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;

final class VeloVoiceOutputVerifier {
    private VeloVoiceOutputVerifier() {}

    static List<Path> collect(Path outputFolder, String base, long started) throws IOException {
        if (!Files.exists(outputFolder)) return List.of();
        String lowerBase = base.toLowerCase();
        try (var stream = Files.walk(outputFolder)) {
            return stream.filter(Files::isRegularFile)
                    .filter(VeloVoiceOutputVerifier::isAudio)
                    .filter(path -> isCurrentOutput(path, lowerBase, started))
                    .sorted()
                    .toList();
        }
    }

    private static boolean isCurrentOutput(Path path, String lowerBase, long started) {
        try {
            String name = path.getFileName().toString().toLowerCase();
            return name.startsWith(lowerBase) && Files.getLastModifiedTime(path).toMillis() >= started - 2000;
        } catch (IOException ignored) {
            return false;
        }
    }

    static boolean isAudio(Path path) {
        String name = path.getFileName().toString().toLowerCase();
        return name.endsWith(".mp3") || name.endsWith(".wav");
    }
}
