package tests

import (
	"context"
	"database/sql"
	"testing"

	"microhabits/internal/db"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := db.Open(context.Background(), "file:test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestOpenCreatesSchema(t *testing.T) {
	database := openTestDB(t)

	var tableCount int
	if err := database.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'table' AND name IN ('users', 'habits', 'habit_completions')
	`).Scan(&tableCount); err != nil {
		t.Fatalf("query schema: %v", err)
	}
	if tableCount != 3 {
		t.Fatalf("expected 3 application tables, got %d", tableCount)
	}
}

func TestSchemaEnforcesRelationshipsAndAllowsMultipleDailyCompletions(t *testing.T) {
	database := openTestDB(t)

	if _, err := database.Exec(`INSERT INTO habits (user_id, name, created_at, updated_at) VALUES (999, 'Read', 1, 1)`); err == nil {
		t.Fatal("expected foreign key violation for an unknown user")
	}

	result, err := database.Exec(`INSERT INTO users (email, password_hash, username, created_at) VALUES ('test@example.com', 'hash', 'janek', 1)`)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID, _ := result.LastInsertId()
	result, err = database.Exec(`INSERT INTO habits (user_id, name, created_at, updated_at) VALUES (?, 'Read', 1, 1)`, userID)
	if err != nil {
		t.Fatalf("insert habit: %v", err)
	}
	habitID, _ := result.LastInsertId()

	if _, err := database.Exec(`INSERT INTO habit_completions (habit_id, completed_at) VALUES (?, 1787529600)`, habitID); err != nil {
		t.Fatalf("insert completion: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO habit_completions (habit_id, completed_at) VALUES (?, 1787533200)`, habitID); err != nil {
		t.Fatalf("insert second same-day completion: %v", err)
	}

	var completionCount int
	if err := database.QueryRow(`SELECT COUNT(*) FROM habit_completions WHERE habit_id = ?`, habitID).Scan(&completionCount); err != nil {
		t.Fatalf("count completions: %v", err)
	}
	if completionCount != 2 {
		t.Fatalf("expected 2 completions, got %d", completionCount)
	}
}