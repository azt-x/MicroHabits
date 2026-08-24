#MicroHabits

Schemat bazy danych sqlite:

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


API

POST /auth/register

body: {
  "email": "test@example.com",
  "password": "password123"
}

POST /auth/login

body: {
  "email": "test@example.com",
  "password": "password123"
}

response: {
    "status": "ok",
    "token": 
}