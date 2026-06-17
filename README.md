# BillShare

A CLI expense sharing application in Go, similar to Splitwise, built using the Bubble Tea TUI framework.

## Features

- **Expense Tracking**: Add expenses with automated equal split calculations.
- **Debt Simplification**: Greedy transaction-minimization engine to simplify transfers.
- **Interactive TUI**: Wizard-driven menus for group creation, expense input, settlements, and member edits.
- **Recalculations**: Dynamic balance updates with manual override commands.
- **Export Formats**:
  - Export a styled PNG report of group expenses, balances, and debts.
  - Generate a formatted text report link copied to the system clipboard for sharing via WhatsApp.
- **Persistence**: Thread-safe local JSON storage layer.

## Architecture

The project follows a clean architectural layout:

- `pkg/domain`: Data structures for User, Group, Expense, and Transfer.
- `pkg/engine`: Balance aggregation and greedy settlement algorithms.
- `pkg/storage`: Thread-safe, file-backed JSON store implementation.
- `pkg/report`: PNG image rendering of reports.
- `pkg/tui`: Bubble Tea terminal interface models and screen update logic.

## Requirements

- Go 1.26 or higher

## Build and Run

### Run Unit Tests
Verify the engine and storage systems:
```bash
go test ./...
```

### Compile the Binary
```bash
go build -o billshare
```

### Start the Application
Run the executable. Data is loaded and saved to `billshare.json` in the working directory:
```bash
./billshare
```

## TUI Keyboard Controls

- **Main Menu**:
  - `u`: Add user
  - `c`: Create group
  - `enter`: Open selected group
  - `q`: Quit

- **Group Details**:
  - `e`: Add new expense (wizard format)
  - `d`: Delete selected expense
  - `s`: Settle outstanding debt (suggests full amount by default, allows partial edits)
  - `a`: Add registered user to the group
  - `o`: Edit who owes (adjust split checkbox list)
  - `r`: Force recalculate balances
  - `p`: Export PNG report image
  - `w`: Copy pre-formatted WhatsApp share link to clipboard
  - `esc` / `b`: Back to main menu
