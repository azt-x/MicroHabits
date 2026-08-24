# MicroHabits

MicroHabits to minimalistyczne API wspierające budowanie pozytywnych, codziennych rutyn.

# Zakres funkcjonalny

Uwierzytelnianie użytkowników (rejestrację i logowanie),
Pełne zarządzanie nawykami (ich tworzenie, podgląd, edycję i usuwanie),
Śledzenie postępów poprzez codzienne odznaczanie zrealizowanych zadań.

# Schemat bazy danych sqlite

users
    id
    email
    password_hash
    username
    created_at

habits
    id
    user_id
    name
    description
    created_at
    updated_at

habit_completions
    id
    habit_id
    completed_date

Relacje:
users 1 - N habits
habits 1 - N habit_completions


# API

POST /auth/register

body: 
{
  "email": "test@example.com",
  "password": "password123"
}

POST /auth/login

body: 
{
  "email": "test@example.com",
  "password": "password123"
}

response: 
{
    "status": "ok",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMSIsImV4cGlyZXNfYXQiOiIxNzg3NTg3MjgzIn0.bTrYxu4vQV8tB8mCqFei6id7FDnN21OO0OsN6Mr8tUk"
}




# JWT

Algorithm & Token Type
{
  "alg": "HS256",
  "typ": "JWT"
}

Payload
{
  "user_id":"1",
  "expires_at":"1787587283"
}