import sys
from PySide6.QtWidgets import QApplication, QWidget, QLabel, QLineEdit, QPushButton, QVBoxLayout, QMessageBox
from PySide6.QtCore import Qt

VERSION = '1.0.1'

class ROICalculator(QWidget):
    def __init__(self):
        super().__init__()
        self.setWindowTitle(f'ROI 计算器 V{VERSION}')
        self.resize(420, 420)

        layout = QVBoxLayout()

        title = QLabel('ROI 投资回报计算器')
        title.setAlignment(Qt.AlignCenter)
        layout.addWidget(title)

        self.cost = QLineEdit()
        self.cost.setPlaceholderText('投入成本')
        layout.addWidget(self.cost)

        self.revenue = QLineEdit()
        self.revenue.setPlaceholderText('收益金额')
        layout.addWidget(self.revenue)

        self.result = QLabel('ROI结果')
        layout.addWidget(self.result)

        btn = QPushButton('计算 ROI')
        btn.clicked.connect(self.calculate)
        layout.addWidget(btn)

        self.setLayout(layout)

    def calculate(self):
        try:
            cost = float(self.cost.text())
            revenue = float(self.revenue.text())
            roi = (revenue - cost) / cost * 100
            self.result.setText(f'利润: {revenue-cost:.2f}\nROI: {roi:.2f}%')
        except Exception:
            QMessageBox.warning(self, '提示', '请输入正确数字')

if __name__ == '__main__':
    app = QApplication(sys.argv)
    win = ROICalculator()
    win.show()
    sys.exit(app.exec())
