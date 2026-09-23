import tkinter as tk
from tkinter import messagebox


class ROICalculator:
    def __init__(self, root):
        self.root = root
        self.root.title('ROI 计算器 V1.0')
        self.root.geometry('420x360')

        tk.Label(root, text='ROI 计算器', font=('Microsoft YaHei', 20)).pack(pady=15)

        self.cost = self.create_input('投资成本')
        self.revenue = self.create_input('收益金额')

        tk.Button(root, text='计算 ROI', command=self.calculate,
                  width=20, height=2).pack(pady=15)

        self.result = tk.Label(root, text='结果将在这里显示', font=('Microsoft YaHei', 12))
        self.result.pack(pady=10)

    def create_input(self, text):
        frame = tk.Frame(self.root)
        frame.pack(pady=5)
        tk.Label(frame, text=text, width=12).pack(side=tk.LEFT)
        entry = tk.Entry(frame, width=25)
        entry.pack(side=tk.LEFT)
        return entry

    def calculate(self):
        try:
            cost = float(self.cost.get())
            revenue = float(self.revenue.get())
            if cost <= 0:
                raise ValueError
            profit = revenue - cost
            roi = profit / cost * 100
            self.result.config(
                text=f'利润: {profit:.2f}\nROI: {roi:.2f}%'
            )
        except Exception:
            messagebox.showerror('错误', '请输入正确数字')


if __name__ == '__main__':
    app = tk.Tk()
    ROICalculator(app)
    app.mainloop()
