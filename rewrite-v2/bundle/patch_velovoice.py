#!/usr/bin/env python3
from pathlib import Path
import shutil, sys
root=Path(sys.argv[1]).resolve()
bundle=Path(sys.argv[2]).resolve()
main=root/'src/main/java/mai_onsyn/VeloVoice/FrameFactory/MainFactory.java'
s=main.read_text(encoding='utf-8')
anchor='    static final AXTextArea textArea = new AXTextArea();\n'
if anchor not in s: raise SystemExit('MainFactory field anchor missing')
s=s.replace(anchor, anchor+'    private static AXDataTreeItem<SimpleStringProperty> automationItem;\n',1)
methods=r'''

    /**
     * RewriteDubbing V2 handoff. Keeps the user's VeloVoice settings untouched and
     * maintains one dedicated visible item for the current automated dubbing task.
     */
    public static AXTreeItem prepareAutomationText(String fileName, String text) {
        if (!Platform.isFxApplicationThread()) throw new IllegalStateException("prepareAutomationText must run on JavaFX thread");
        String safeName = (fileName == null || fileName.isBlank()) ? "RewriteDubbing" : fileName;
        String value = text == null ? "" : text;
        if (automationItem == null || automationItem.getParentItem() == null) {
            automationItem = treeView.createFileItem(safeName, new SimpleStringProperty(""));
            treeView.add(treeView.getRoot(), automationItem);
        } else {
            automationItem.rename(safeName);
        }
        treeView.getGroup().selectButton(automationItem.getButton());
        if (automationItem.getData() instanceof SimpleStringProperty property) {
            property.set("");
            textArea.setText("");
            property.set(value);
            textArea.setText(value);
        }
        treeView.update();
        return automationItem;
    }

    public static String getAutomationEditorText() {
        if (!Platform.isFxApplicationThread()) throw new IllegalStateException("getAutomationEditorText must run on JavaFX thread");
        return textArea.getText() == null ? "" : textArea.getText();
    }

    public static AXTreeItem getAutomationItem() {
        return automationItem;
    }

    public static File getConfiguredAudioSaveFolder() {
        return new File(config.getString("AudioSaveFolder"));
    }
'''
idx=s.rfind('\n}')
if idx<0: raise SystemExit('MainFactory final brace missing')
s=s[:idx]+methods+s[idx:]
main.write_text(s,encoding='utf-8')


# Store VeloVoice config in the path selected by RewriteDubbing V2 instead of the install directory.
runtime=root/'src/main/java/mai_onsyn/VeloVoice/App/Runtime.java'
rs=runtime.read_text(encoding='utf-8')
lines=rs.splitlines(keepends=True)
anchor_index=next((i for i,line in enumerate(lines) if 'private static final File CONFIG_FILE = new File(System.getProperty("user.dir")' in line), None)
if anchor_index is None: raise SystemExit('Runtime CONFIG_FILE anchor missing')
replacement='    private static final File CONFIG_FILE = resolveConfigFile();\n    private static File resolveConfigFile() {\n        String configured = System.getProperty("rewrite.dubbing.velovoice.config", "").trim();\n        File file = configured.isBlank() ? new File(System.getProperty("user.dir"), "config.json") : new File(configured);\n        File parent = file.getParentFile();\n        if (parent != null) parent.mkdirs();\n        return file;\n    }\n'
lines[anchor_index]=replacement
rs=''.join(lines)
runtime.write_text(rs,encoding='utf-8')

src=bundle/'v2-sidecar-src/RewriteDubbingV2/velo'
dst=root/'src/main/java/mai_onsyn/RewriteDubbingV2/velo'
dst.mkdir(parents=True,exist_ok=True)
for f in src.glob('*.java'):
    shutil.copy2(f,dst/f.name)
print('VELOVOICE_V2_PATCH_APPLIED')
