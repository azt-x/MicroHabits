# MicroHabits

MicroHabits to minimalistyczne API wspierające budowanie pozytywnych, codziennych rutyn.

## Zakres funkcjonalny

- Uwierzytelnianie użytkowników (rejestrację i logowanie)
- Pełne zarządzanie nawykami (ich tworzenie, podgląd, edycję i usuwanie)
- Śledzenie postępów poprzez codzienne odznaczanie zrealizowanych zadań

## Schemat bazy danych sqlite

users

- id
- email
- password_hash
- username
- created_at

habits

- id
- user_id
- name
- description
- created_at
- updated_at

habit_completions

- id
- habit_id
- completed_date

Relacje:

- users 1 - N habits
- habits 1 - N habit_completions

## API

### Konwencje odpowiedzi

- Wszystkie poprawne odpowiedzi zwracają pole `status: "ok"`
- Wszystkie błędy zwracają pole `status: "error"`
- Kod HTTP odpowiedzi powinien odzwierciedlać typ błędu lub sukcesu
- Dla kolekcji używany jest układ `items` + meta `page`, `limit`, `total`

Przykład błędu:

```json
{
  "status": "error",
  "code": "VALIDATION_ERROR",
  "message": "Email jest wymagany",
  "details": {
    "email": ["required"]
  }
}
```

### POST /auth/register

Rejestracja nowego użytkownika.

body:

```json
{
  "email": "test@example.com",
  "username": "janek",
  "password": "password123"
}
```

response: `201 Created`

```json
{
  "status": "ok",
  "user": {
    "id": 1,
    "email": "test@example.com",
    "username": "janek"
  }
}
```

Błędy:

- `409 CONFLICT` z `code: "EMAIL_ALREADY_EXISTS"` jeżeli email jest zajęty
- `422 UNPROCESSABLE_ENTITY` z `code: "VALIDATION_ERROR"` przy błędnych danych wejściowych

### POST /auth/login

body:

```json
{
  "email": "test@example.com",
  "password": "password123"
}
```

response: `200 OK`

```json
{
  "status": "ok",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMSIsImV4cGlyZXNfYXQiOiIxNzg3NTg3MjgzIn0.bTrYxu4vQV8tB8mCqFei6id7FDnN21OO0OsN6Mr8tUk",
  "user": {
    "id": 1,
    "email": "test@example.com",
    "username": "janek"
  }
}
```

Błędy:

- `401 UNAUTHORIZED` z `code: "INVALID_CREDENTIALS"`
- `429 TOO_MANY_REQUESTS` z `code: "RATE_LIMITED"` przy zbyt wielu próbach logowania

### GET /me

Zwraca aktualnie zalogowanego użytkownika.

response: `200 OK`

```json
{
  "status": "ok",
  "user": {
    "id": 1,
    "email": "test@example.com",
    "username": "janek",
    "created_at": 1787587283
  }
}
```

## Autoryzacja

Wszystkie endpointy poza `/auth/register` oraz `/auth/login` wymagają nagłówka:

```text
Authorization: Bearer <token>
```

W przypadku braku tokenu, niepoprawnego tokenu lub wygasłego tokenu API zwraca:

```text
401 Unauthorized
```

z payloadem:

```json
{
  "status": "error",
  "code": "INVALID_TOKEN",
  "message": "Token is missing or invalid"
}
```

## Globalne błędy

Wszystkie błędy w API zwracane są w jednym formacie:

```json
{
  "status": "error",
  "code": "VALIDATION_ERROR",
  "message": "Nieprawidłowe dane wejściowe",
  "details": {}
}
```

Najczęstsze kody błędów:

- `400 BAD_REQUEST` - niepoprawne parametry zapytania lub body
- `401 UNAUTHORIZED` - brak lub niepoprawny token
- `403 FORBIDDEN` - brak dostępu do zasobu
- `404 NOT_FOUND` - zasób nie istnieje
- `409 CONFLICT` - konflikt stanu danych, np. duplikat emaila lub powtórne zaznaczenie tego samego dnia
- `422 UNPROCESSABLE_ENTITY` - poprawny JSON, ale błędne dane biznesowe
- `429 TOO_MANY_REQUESTS` - przekroczony limit żądań
- `500 INTERNAL_SERVER_ERROR` - nieoczekiwany błąd po stronie serwera

## Zarządzanie nawykami

### GET /habits

query params:

- `page` (optional, integer, default: 1, min: 1)
- `limit` (optional, integer, default: 20, min: 1, max: 100)
- `sort` (optional, string, default: `created_at`)
- `order` (optional, asc|desc, default: desc)

response: `200 OK`

```json
{
  "status": "ok",
  "page": 1,
  "limit": 20,
  "total": 42,
  "items": [
    {
      "id": 1,
      "user_id": 1,
      "name": "Czytanie",
      "description": "Przeczytać 10 stron książki",
      "created_at": 1787587283,
      "updated_at": 1787587283
    }
  ]
}
```

### POST /habits

body:

```json
{
  "name": "Czytanie",
  "description": "Przeczytać 10 stron książki"
}
```

response: `201 Created`

```json
{
  "status": "ok",
  "habit": {
    "id": 1,
    "user_id": 1,
    "name": "Czytanie",
    "description": "Przeczytać 10 stron książki",
    "created_at": 1787587283,
    "updated_at": 1787587283
  }
}
```

Błędy:

- `400 BAD_REQUEST` z `code: "INVALID_HABIT_NAME"` jeżeli nazwa jest pusta
- `422 UNPROCESSABLE_ENTITY` z `code: "VALIDATION_ERROR"` przy niepoprawnych danych

### GET /habits/{id}

response: `200 OK`

```json
{
  "status": "ok",
  "habit": {
    "id": 1,
    "user_id": 1,
    "name": "Czytanie",
    "description": "Przeczytać 10 stron książki",
    "created_at": 1787587283,
    "updated_at": 1787587283
  }
}
```

Błędy:

- `404 NOT_FOUND` z `code: "HABIT_NOT_FOUND"`
- `403 FORBIDDEN` z `code: "FORBIDDEN"` jeżeli habit nie należy do zalogowanego użytkownika

### PUT /habits/{id}

body:

```json
{
  "name": "Czytanie",
  "description": "Przeczytać 20 stron książki"
}
```

response: `200 OK`

```json
{
  "status": "ok",
  "habit": {
    "id": 1,
    "user_id": 1,
    "name": "Czytanie",
    "description": "Przeczytać 20 stron książki",
    "created_at": 1787587283,
    "updated_at": 1787587283
  }
}
```

### DELETE /habits/{id}

response: `200 OK`

```json
{
  "status": "ok",
  "message": "Habit deleted"
}
```

Błędy:

- `404 NOT_FOUND` z `code: "HABIT_NOT_FOUND"`
- `403 FORBIDDEN` z `code: "FORBIDDEN"`

### GET /habits/{id}/completed

query params:

- `page` (optional, integer, default: 1, min: 1)
- `limit` (optional, integer, default: 30, min: 1, max: 365)
- `order` (optional, asc|desc, default: desc)

response: `200 OK`

```json
{
  "status": "ok",
  "habit_id": 1,
  "page": 1,
  "limit": 30,
  "total": 3,
  "order": "desc",
  "items": [
    {
      "id": 10,
      "habit_id": 1,
      "completed_date": 1787587283
    },
    {
      "id": 9,
      "habit_id": 1,
      "completed_date": 1787500883
    },
    {
      "id": 8,
      "habit_id": 1,
      "completed_date": 1787414483
    }
  ]
}
```

### POST /habits/{id}/completed

Dodaje realizację zadania.

response: `201 Created`

```json
{
  "status": "ok",
  "completion": {
    "id": 10,
    "habit_id": 1,
    "completed_date": 1787587283
  }
}
```

Błędy:

- `404 NOT_FOUND` z `code: "HABIT_NOT_FOUND"`


### DELETE /habits/{id}/completed/{completionId}

Usuwa konkretny wpis o ukończeniu nawyku.

response: `200 OK`

```json
{
  "status": "ok",
  "message": "Completion deleted"
}
```

Błędy:

- `404 NOT_FOUND` z `code: "COMPLETION_NOT_FOUND"`
- `403 FORBIDDEN` z `code: "FORBIDDEN"`

## JWT

Algorithm & Token Type

```json
{
  "alg": "HS256",
  "typ": "JWT"
}
```

Payload

```json
{
  "user_id": 1,
  "exp": 1787587283
}
```

Encoded token:

```text
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMSIsImV4cCI6MTc4NzU4NzI4M30.bTrYxu4vQV8tB8mCqFei6id7FDnN21OO0OsN6Mr8tUk
```