@echo off
python -m pip install pyinstaller
pyinstaller --onefile --windowed roi_calculator.py
pause
