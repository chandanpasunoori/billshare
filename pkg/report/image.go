package report

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"time"

	"billshare/pkg/domain"
	"billshare/pkg/engine"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// GenerateGroupReportImage creates a styled PNG report for a group and saves it to the given path.
func GenerateGroupReportImage(g domain.Group, allUsers []domain.User, outputPath string) error {
	// 1. Calculate balances and simplified debts
	balances := engine.CalculateBalances(g.Members, g.Expenses)
	transfers := engine.SettleDebts(balances)

	// Map user IDs to names for easy display
	userMap := make(map[string]string)
	for _, u := range allUsers {
		userMap[u.ID] = u.Name
	}
	getUserName := func(id string) string {
		if name, ok := userMap[id]; ok {
			return name
		}
		return id
	}

	// 2. Calculate dynamic height based on data
	numExpenses := len(g.Expenses)
	numMembers := len(g.Members)
	numTransfers := len(transfers)

	// Base height + dynamic height
	leftHeight := 80 + (numExpenses * 22)
	rightHeight := 100 + (numMembers * 22) + (numTransfers * 22)
	
	maxContentHeight := leftHeight
	if rightHeight > leftHeight {
		maxContentHeight = rightHeight
	}
	if maxContentHeight < 300 {
		maxContentHeight = 300
	}
	
	width := 850
	height := maxContentHeight + 100

	// 3. Define Color Palette (Sleek Dark Theme)
	bgColor := color.RGBA{0x0f, 0x17, 0x2a, 0xff}      // Slate 900
	primaryColor := color.RGBA{0x7d, 0x56, 0xf4, 0xff}   // Violet
	textColor := color.RGBA{0xc1, 0xc6, 0xe2, 0xff}      // Pastel Gray
	dimColor := color.RGBA{0x62, 0x68, 0x8f, 0xff}       // Muted Gray-Blue
	greenColor := color.RGBA{0x04, 0xb5, 0x75, 0xff}     // Emerald Green
	redColor := color.RGBA{0xff, 0x4c, 0x54, 0xff}       // Coral Red
	whiteColor := color.RGBA{0xff, 0xff, 0xff, 0xff}

	// Create blank image
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// Helper function to draw text
	drawText := func(x, y int, txt string, col color.Color) {
		point := fixed.Point26_6{X: fixed.Int26_6(x << 6), Y: fixed.Int26_6(y << 6)}
		d := &font.Drawer{
			Dst:  img,
			Src:  image.NewUniform(col),
			Face: basicfont.Face7x13,
			Dot:  point,
		}
		d.DrawString(txt)
	}

	// Helper to draw horizontal line
	drawLine := func(x1, y1, x2, y2 int, col color.Color) {
		for x := x1; x <= x2; x++ {
			img.Set(x, y1, col)
		}
	}

	// Helper to format currency
	formatCents := func(cents int64) string {
		dollars := cents / 100
		remCents := cents % 100
		if remCents < 0 {
			remCents = -remCents
		}
		sign := ""
		if cents < 0 {
			sign = "-"
			if dollars < 0 {
				dollars = -dollars
			}
		}
		return fmt.Sprintf("%s$%d.%02d", sign, dollars, remCents)
	}

	// --- Draw Header ---
	drawText(25, 35, "BILLSHARE REPORT", primaryColor)
	drawText(25, 55, fmt.Sprintf("Group: %s", g.Name), whiteColor)
	
	// Draw timestamp
	timeStr := time.Now().Format("Jan 02, 2006 15:04:05")
	drawText(width-200, 35, fmt.Sprintf("Generated: %s", timeStr), dimColor)

	drawLine(25, 70, width-25, 70, dimColor)

	// --- Left Column: Expenses ---
	yLeft := 100
	drawText(25, yLeft, "EXPENSE HISTORY", primaryColor)
	drawLine(25, yLeft+5, 400, yLeft+5, dimColor)
	yLeft += 25

	if len(g.Expenses) == 0 {
		drawText(25, yLeft, "No expenses recorded yet.", dimColor)
	} else {
		for _, exp := range g.Expenses {
			payer := getUserName(exp.PaidBy)
			expStr := fmt.Sprintf("%-20s %9s (by %s)", truncateString(exp.Description, 20), formatCents(exp.Amount), payer)
			drawText(25, yLeft, expStr, textColor)
			yLeft += 22
		}
	}

	// --- Right Column: Balances & Simplified Debts ---
	yRight := 100
	drawText(450, yRight, "NET BALANCES", primaryColor)
	drawLine(450, yRight+5, width-25, yRight+5, dimColor)
	yRight += 25

	for _, mID := range g.Members {
		bal := balances[mID]
		name := getUserName(mID)
		
		balStr := formatCents(bal)
		var balCol color.Color = textColor
		if bal > 0 {
			balStr = "+" + balStr
			balCol = greenColor
		} else if bal < 0 {
			balCol = redColor
		}

		drawText(450, yRight, fmt.Sprintf("%-20s", name), textColor)
		drawText(620, yRight, balStr, balCol)
		yRight += 22
	}

	yRight += 15
	drawText(450, yRight, "SIMPLIFIED DEBTS", primaryColor)
	drawLine(450, yRight+5, width-25, yRight+5, dimColor)
	yRight += 25

	if len(transfers) == 0 {
		drawText(450, yRight, "All settled up!", greenColor)
	} else {
		for _, tr := range transfers {
			fromName := getUserName(tr.From)
			toName := getUserName(tr.To)
			transferStr := fmt.Sprintf("%s owes %s %s", fromName, toName, formatCents(tr.Amount))
			drawText(450, yRight, transferStr, textColor)
			yRight += 22
		}
	}

	// --- Footer ---
	drawLine(25, height-35, width-25, height-35, dimColor)
	drawText(25, height-20, "BillShare - Minimal Expense Sharing TUI", dimColor)

	// Save file
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

func truncateString(s string, l int) string {
	if len(s) <= l {
		return s
	}
	return s[:l-3] + "..."
}
