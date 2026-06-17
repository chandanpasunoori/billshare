package report

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chandanpasunoori/billshare/pkg/domain"
)

func TestGenerateGroupReportImage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "billshare_report_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	outputPath := filepath.Join(tempDir, "report.png")

	g := domain.Group{
		ID:      "g1",
		Name:    "Summer Trip",
		Members: []string{"u1", "u2"},
		Expenses: []domain.Expense{
			{
				ID:          "e1",
				Description: "Cabin rental",
				Amount:      50000, // $500.00
				PaidBy:      "u1",
				Splits: map[string]int64{
					"u1": 25000,
					"u2": 25000,
				},
				Date: time.Now(),
			},
		},
	}

	allUsers := []domain.User{
		{ID: "u1", Name: "Alice"},
		{ID: "u2", Name: "Bob"},
	}

	err = GenerateGroupReportImage(g, allUsers, outputPath)
	if err != nil {
		t.Fatalf("failed to generate report image: %v", err)
	}

	// Verify file exists and is not empty
	fi, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("output file does not exist: %v", err)
	}
	if fi.Size() == 0 {
		t.Errorf("output file is empty")
	}
}
