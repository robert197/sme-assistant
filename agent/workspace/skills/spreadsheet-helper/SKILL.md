---
name: spreadsheet-helper
description: Create, read, and manipulate Excel spreadsheets and CSV files
---

# Spreadsheet Helper

Work with Excel (.xlsx) and CSV files using Python and openpyxl.

## When to use

- User asks to create a spreadsheet or Excel file
- User asks to read, summarize, or analyze data from .xlsx or .csv files
- User asks to add rows, columns, or formulas to an existing spreadsheet
- User mentions "spreadsheet", "Excel", "CSV", or "table data"

## How to use

Use the `exec` tool to run Python scripts with the `openpyxl` library (pre-installed).

### Read a spreadsheet

```python
import openpyxl
wb = openpyxl.load_workbook("/workspace/example.xlsx")
ws = wb.active
for row in ws.iter_rows(values_only=True):
    print(row)
```

### Create a new spreadsheet

```python
import openpyxl
wb = openpyxl.Workbook()
ws = wb.active
ws.title = "Sheet1"
ws.append(["Name", "Email", "Role"])
ws.append(["Alice", "alice@example.com", "Manager"])
wb.save("/workspace/output.xlsx")
print("Saved to /workspace/output.xlsx")
```

### Read a CSV file

```python
import csv
with open("/workspace/data.csv") as f:
    reader = csv.DictReader(f)
    for row in reader:
        print(row)
```

## Important

- Always save output files to `/workspace/` so the user can access them
- Show a summary of the data after reading (row count, column names, first few rows)
- When creating spreadsheets, confirm the filename and columns with the user first
