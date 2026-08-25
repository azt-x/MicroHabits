package tests

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"microhabits/internal/auth"
	"microhabits/internal/db"

	"github.com/golang-jwt/jwt/v5"
)

func testService(t *testing.T) *auth.Service {
	t.Helper()
	database, err := db.Open(context.Background(), "file:auth-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return auth.NewService(database, "test-secret")
}

func TestRegisterHashesPasswordAndLoginReturnsToken(t *testing.T) {
	service := testService(t)
	user, err := service.Register(context.Background(), " TEST@example.com ", "janek", "password123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Email != "test@example.com" {
		t.Fatalf("expected normalized email, got %q", user.Email)
	}

	loggedInUser, token, err := service.Login(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if loggedInUser.ID != user.ID || token == "" {
		t.Fatalf("unexpected login result: user=%+v token=%q", loggedInUser, token)
	}

	if _, _, err := service.Login(context.Background(), "test@example.com", "wrong-password"); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	service := testService(t)
	if _, err := service.Register(context.Background(), "test@example.com", "first", "password123"); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if _, err := service.Register(context.Background(), "TEST@example.com", "second", "password123"); !errors.Is(err, auth.ErrEmailExists) {
		t.Fatalf("expected duplicate email error, got %v", err)
	}
}

func TestRegisterRejectsInvalidEmailOrPassword(t *testing.T) {
	service := testService(t)

	if _, err := service.Register(context.Background(), "invalid-email", "janek", "password123"); !errors.Is(err, auth.ErrValidation) {
		t.Fatalf("expected validation error for invalid email, got %v", err)
	}

	if _, err := service.Register(context.Background(), "test2@example.com", "janek", "password"); !errors.Is(err, auth.ErrValidation) {
		t.Fatalf("expected validation error for password without digit, got %v", err)
	}

	if _, err := service.Register(context.Background(), "test3@example.com", "janek", "12345678"); !errors.Is(err, auth.ErrValidation) {
		t.Fatalf("expected validation error for password without letter, got %v", err)
	}
}

func TestAuthHandlers(t *testing.T) {
	service := testService(t)
	handler := auth.NewHandler(service)

	registerRequest := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"email":"test@example.com","username":"janek","password":"password123"}`))
	registerResponse := httptest.NewRecorder()
	handler.Register(registerResponse, registerRequest)
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", registerResponse.Code)
	}

	loginRequest := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"test@example.com","password":"password123"}`))
	loginResponse := httptest.NewRecorder()
	handler.Login(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK || !strings.Contains(loginResponse.Body.String(), `"token"`) {
		t.Fatalf("unexpected login response: status=%d body=%s", loginResponse.Code, loginResponse.Body.String())
	}
}

func TestExpiredTokenIsRejected(t *testing.T) {
	service := testService(t)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 1,
		"exp":     time.Now().Add(-time.Minute).Unix(),
	})
	encoded, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/me", nil)
	request.Header.Set("Authorization", "Bearer "+encoded)
	response := httptest.NewRecorder()
	service.AuthenticateRequestForHandler(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "TOKEN_EXPIRED") {
		t.Fatalf("expected expired token code in response, got %s", response.Body.String())
	}
}

func TestAuthResponseAndTokenPayloadOrder(t *testing.T) {
	service := testService(t)
	if _, err := service.Register(context.Background(), "order@example.com", "janek", "password123"); err != nil {
		t.Fatalf("register: %v", err)
	}

	handler := auth.NewHandler(service)
	request := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"order@example.com","password":"password123"}`))
	response := httptest.NewRecorder()
	handler.Login(response, request)
	body := response.Body.String()
	if strings.Index(body, `"status"`) > strings.Index(body, `"token"`) {
		t.Fatalf("response fields are not ordered as status -> token -> user: %s", body)
	}
	if strings.Index(body, `"token"`) > strings.Index(body, `"user"`) {
		t.Fatalf("response fields are not ordered as status -> token -> user: %s", body)
	}

	parts := strings.Split(response.Body.String(), `"token":"`)
	if len(parts) < 2 {
		t.Fatalf("token missing from response: %s", response.Body.String())
	}
	tokenValue := strings.SplitN(parts[1], `"`, 2)[0]
	payload := strings.Split(tokenValue, ".")[1]
	decodedBytes, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("decode token payload: %v", err)
	}
	decoded := string(decodedBytes)
	if strings.Index(decoded, `"user_id"`) > strings.Index(decoded, `"exp"`) {
		t.Fatalf("token claims are not ordered as user_id -> exp: %s", decoded)
	}
}
