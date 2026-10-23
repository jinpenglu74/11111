package mai_onsyn.RewriteDubbingV2.velo;

import java.nio.file.Path;

public final class VeloVoiceProtocolSelfTest {
    public static void main(String[] args) {
        int passed = 0;
        check(VeloVoiceAutomation.sameText("a\r\nb", "a\nb"), "newline canonicalization"); passed++;
        check(!VeloVoiceAutomation.sameText(" a b ", "a b"), "leading trailing spaces preserved"); passed++;
        check(!VeloVoiceAutomation.sameText("a  b", "a b"), "double spaces preserved"); passed++;
        check(VeloVoiceAutomation.isAudio(Path.of("x.MP3")), "mp3 filter"); passed++;
        check(VeloVoiceAutomation.isAudio(Path.of("x.wav")), "wav filter"); passed++;
        check(!VeloVoiceAutomation.isAudio(Path.of("x.txt")), "non audio filter"); passed++;
        check("movie_01".equals(VeloVoiceAutomation.safeBase("movie:01.srt")), "safe base"); passed++;
        System.out.println("VELOVOICE_V2_PROTOCOL_PASS " + passed + "/7");
    }
    private static void check(boolean ok, String label) { if (!ok) throw new AssertionError(label); }
}
