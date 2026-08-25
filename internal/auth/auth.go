package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists        = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrValidation         = errors.New("validation error")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	emailRegex            = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	passwordLetterRegex   = regexp.MustCompile(`[A-Za-z]`)
	passwordDigitRegex    = regexp.MustCompile(`\d`)
)

type User struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	CreatedAt int64  `json:"created_at"`
}

type Service struct {
	database  *sql.DB
	jwtSecret []byte
	tokenTTL  time.Duration
	now       func() time.Time
}

func NewService(database *sql.DB, jwtSecret string) *Service {
	return &Service{
		database:  database,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  24 * time.Hour,
		now:       time.Now,
	}
}

func (service *Service) Register(ctx context.Context, email, username, password string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	username = strings.TrimSpace(username)
	if email == "" || username == "" || !isValidEmail(email) || !isValidPassword(password) {
		return User{}, ErrValidation
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}

	createdAt := service.now().Unix()
	result, err := service.database.ExecContext(ctx, `
		INSERT INTO users (email, password_hash, username, created_at)
		VALUES (?, ?, ?, ?)
	`, email, string(passwordHash), username, createdAt)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return User{}, ErrEmailExists
		}
		return User{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return User{}, err
	}
	return User{ID: id, Email: email, Username: username, CreatedAt: createdAt}, nil
}

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func isValidPassword(password string) bool {
	if len(password) < 8 || len(password) > 72 {
		return false
	}
	return passwordLetterRegex.MatchString(password) && passwordDigitRegex.MatchString(password)
}

type TokenClaims struct {
	UserID int64 `json:"user_id"`
	Exp    int64 `json:"exp"`
}

func (claims TokenClaims) GetAudience() (jwt.ClaimStrings, error) {
	return nil, nil
}

func (claims TokenClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	if claims.Exp == 0 {
		return nil, nil
	}
	return jwt.NewNumericDate(time.Unix(claims.Exp, 0)), nil
}

func (claims TokenClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	return nil, nil
}

func (claims TokenClaims) GetIssuer() (string, error) {
	return "", nil
}

func (claims TokenClaims) GetNotBefore() (*jwt.NumericDate, error) {
	return nil, nil
}

func (claims TokenClaims) GetSubject() (string, error) {
	return "", nil
}

func (claims TokenClaims) Valid() error {
	if claims.Exp > 0 && time.Now().Unix() >= claims.Exp {
		return jwt.ErrTokenExpired
	}
	return nil
}

func (service *Service) Login(ctx context.Context, email, password string) (User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var user User
	var passwordHash string
	if err := service.database.QueryRowContext(ctx, `
		SELECT id, email, username, password_hash, created_at
		FROM users WHERE email = ?
	`, email).Scan(&user.ID, &user.Email, &user.Username, &passwordHash, &user.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, "", ErrInvalidCredentials
		}
		return User{}, "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return User{}, "", ErrInvalidCredentials
	}

	claims := TokenClaims{
		UserID: user.ID,
		Exp:    service.now().Add(service.tokenTTL).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	encodedToken, err := token.SignedString(service.jwtSecret)
	if err != nil {
		return User{}, "", err
	}
	return user, encodedToken, nil
}

func (service *Service) UserByID(ctx context.Context, userID int64) (User, error) {
	var user User
	if err := service.database.QueryRowContext(ctx, `
		SELECT id, email, username, created_at
		FROM users WHERE id = ?
	`, userID).Scan(&user.ID, &user.Email, &user.Username, &user.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrInvalidToken
		}
		return User{}, err
	}
	return user, nil
}

func (service *Service) ParseToken(tokenString string) (jwt.MapClaims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, ErrInvalidToken
	}
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return service.jwtSecret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (service *Service) AuthenticateRequest(request *http.Request) (User, error) {
	header := request.Header.Get("Authorization")
	if header == "" {
		return User{}, ErrInvalidToken
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return User{}, ErrInvalidToken
	}
	claims, err := service.ParseToken(parts[1])
	if err != nil {
		return User{}, err
	}
	userIDValue, ok := claims["user_id"]
	if !ok {
		return User{}, ErrInvalidToken
	}
	var userID int64
	switch value := userIDValue.(type) {
	case float64:
		userID = int64(value)
	case int64:
		userID = value
	case int:
		userID = int64(value)
	case jsonNumber:
		userID = int64(value)
	default:
		return User{}, ErrInvalidToken
	}
	return service.UserByID(request.Context(), userID)
}

func (service *Service) AuthenticateRequestForHandler(w http.ResponseWriter, request *http.Request) (User, bool) {
	user, err := service.AuthenticateRequest(request)
	if err != nil {
		switch {
		case errors.Is(err, ErrTokenExpired):
			writeError(w, http.StatusUnauthorized, "TOKEN_EXPIRED", "Token has expired")
		default:
			writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Token is missing or invalid")
		}
		return User{}, false
	}
	return user, true
}

type jsonNumber int64
