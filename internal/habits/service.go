package habits

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrHabitNotFound      = errors.New("habit not found")
	ErrCompletionNotFound = errors.New("completion not found")
	ErrForbidden          = errors.New("forbidden")
	ErrValidation         = errors.New("validation error")
)

type Habit struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

type Completion struct {
	ID          int64 `json:"id"`
	HabitID     int64 `json:"habit_id"`
	CompletedAt int64 `json:"completed_at"`
}

type Service struct {
	database *sql.DB
	now      func() time.Time
}

func NewService(database *sql.DB) *Service {
	return &Service{database: database, now: time.Now}
}

func (service *Service) ListHabits(ctx context.Context, userID int64, page, limit int, sort, order string) ([]Habit, int, error) {
	page = normalizePage(page)
	limit = normalizeLimit(limit)
	sort = normalizeSort(sort)
	order = normalizeOrder(order)

	var total int
	if err := service.database.QueryRowContext(ctx, `SELECT COUNT(*) FROM habits WHERE user_id = ?`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, name, description, created_at, updated_at
		FROM habits
		WHERE user_id = ?
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, sort, order)
	rows, err := service.database.QueryContext(ctx, query, userID, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Habit, 0, limit)
	for rows.Next() {
		var habit Habit
		if err := rows.Scan(&habit.ID, &habit.UserID, &habit.Name, &habit.Description, &habit.CreatedAt, &habit.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, habit)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (service *Service) CreateHabit(ctx context.Context, userID int64, name, description string) (Habit, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" {
		return Habit{}, ErrValidation
	}
	createdAt := service.now().Unix()
	result, err := service.database.ExecContext(ctx, `
		INSERT INTO habits (user_id, name, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, userID, name, description, createdAt, createdAt)
	if err != nil {
		return Habit{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Habit{}, err
	}
	return Habit{ID: id, UserID: userID, Name: name, Description: description, CreatedAt: createdAt, UpdatedAt: createdAt}, nil
}

func (service *Service) GetHabit(ctx context.Context, userID, habitID int64) (Habit, error) {
	var habit Habit
	if err := service.database.QueryRowContext(ctx, `
		SELECT id, user_id, name, description, created_at, updated_at
		FROM habits
		WHERE id = ?
	`, habitID).Scan(&habit.ID, &habit.UserID, &habit.Name, &habit.Description, &habit.CreatedAt, &habit.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Habit{}, ErrHabitNotFound
		}
		return Habit{}, err
	}
	if habit.UserID != userID {
		return Habit{}, ErrForbidden
	}
	return habit, nil
}

func (service *Service) UpdateHabit(ctx context.Context, userID, habitID int64, name, description string) (Habit, error) {
	habit, err := service.GetHabit(ctx, userID, habitID)
	if err != nil {
		return Habit{}, err
	}
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" {
		return Habit{}, ErrValidation
	}
	habit.Name = name
	habit.Description = description
	habit.UpdatedAt = service.now().Unix()
	result, err := service.database.ExecContext(ctx, `
		UPDATE habits SET name = ?, description = ?, updated_at = ? WHERE id = ?
	`, habit.Name, habit.Description, habit.UpdatedAt, habitID)
	if err != nil {
		return Habit{}, err
	}
	if rows, err := result.RowsAffected(); err != nil || rows == 0 {
		return Habit{}, ErrHabitNotFound
	}
	return habit, nil
}

func (service *Service) DeleteHabit(ctx context.Context, userID, habitID int64) error {
	if _, err := service.GetHabit(ctx, userID, habitID); err != nil {
		return err
	}
	result, err := service.database.ExecContext(ctx, `DELETE FROM habits WHERE id = ?`, habitID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrHabitNotFound
	}
	return nil
}

func (service *Service) ListCompletions(ctx context.Context, userID, habitID int64, page, limit int, order string) ([]Completion, int, error) {
	if _, err := service.GetHabit(ctx, userID, habitID); err != nil {
		return nil, 0, err
	}
	page = normalizePage(page)
	limit = normalizeLimit(limit)
	order = normalizeOrder(order)

	var total int
	if err := service.database.QueryRowContext(ctx, `SELECT COUNT(*) FROM habit_completions WHERE habit_id = ?`, habitID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := service.database.QueryContext(ctx, `
		SELECT id, habit_id, completed_at
		FROM habit_completions
		WHERE habit_id = ?
		ORDER BY completed_at `+order+` 
		LIMIT ? OFFSET ?
	`, habitID, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Completion, 0, limit)
	for rows.Next() {
		var completion Completion
		if err := rows.Scan(&completion.ID, &completion.HabitID, &completion.CompletedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, completion)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (service *Service) CreateCompletion(ctx context.Context, userID, habitID int64) (Completion, error) {
	if _, err := service.GetHabit(ctx, userID, habitID); err != nil {
		return Completion{}, err
	}
	completedAt := service.now().Unix()
	result, err := service.database.ExecContext(ctx, `
		INSERT INTO habit_completions (habit_id, completed_at)
		VALUES (?, ?)
	`, habitID, completedAt)
	if err != nil {
		return Completion{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Completion{}, err
	}
	return Completion{ID: id, HabitID: habitID, CompletedAt: completedAt}, nil
}

func (service *Service) DeleteCompletion(ctx context.Context, userID, habitID, completionID int64) error {
	if _, err := service.GetHabit(ctx, userID, habitID); err != nil {
		return err
	}
	var exists bool
	if err := service.database.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM habit_completions WHERE id = ? AND habit_id = ?)`, completionID, habitID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrCompletionNotFound
	}
	result, err := service.database.ExecContext(ctx, `DELETE FROM habit_completions WHERE id = ? AND habit_id = ?`, completionID, habitID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrCompletionNotFound
	}
	return nil
}

func normalizePage(page int) int {
	if page <= 0 {
		return 1
	}
	return page
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func normalizeSort(sort string) string {
	switch strings.TrimSpace(strings.ToLower(sort)) {
	case "updated_at":
		return "updated_at"
	case "name":
		return "name"
	default:
		return "created_at"
	}
}

func normalizeOrder(order string) string {
	switch strings.TrimSpace(strings.ToLower(order)) {
	case "asc":
		return "ASC"
	default:
		return "DESC"
	}
}
