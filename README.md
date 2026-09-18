# stuck-json ⚡

> **Fast, Ergonomic, Zero-Boilerplate JSON Toolkit for Go 1.22+**  
> Высокопроизводительная библиотека для Go, объединяющая декодирование на дженериках, быструю валидацию по тегам, автоматическую подстановку значений по умолчанию (defaults) и безопасный биндинг HTTP-запросов.

[![Go Reference](https://pkg.go.dev/badge/github.com/s7ukk/stuck-json.svg)](https://pkg.go.dev/github.com/s7ukk/stuck-json)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/s7ukk/stuck-json)](https://goreportcard.com/report/github.com/s7ukk/stuck-json)

---

## Почему stuck-json ?

Стандартный пакет Go `encoding/json` требует много шаблонного кода: ручная проверка типов, ручная валидация каждого поля, обработка ошибок, проверка заголовков HTTP, лимит размера `r.Body` для защиты от DoS, и заполнение дефолтных значений.

**`stuck-json` решает это в 1 строчку:**
1. **Лаконичный код**: вместо 30-40 строк boilerplate — одна строчка: `req, err := stuckjson.BindAndValidate[CreateUserRequest](r)`.
2. **Нулевые тяжелые зависимости (Zero external bloat)**: библиотека использует оптимизированный стандартный рантайм Go, кэширует рефлексию и не тянет сторонних пакетов.
3. **Автоподгрузка того, что не установлено (`default:"..."`)**: если клиент не передал необязательные поля в JSON, `stuck-json` сам подставит дефолтные значения (строки, числа, булевы флаги, слайсы, `default:"now"` или `default:"uuid"`).
4. **Умная валидация (`validate:"..."`)**: проверяет поля по тегам, сопоставляет ошибки с реальными JSON-именами (`json:"user_email"` -> ошибка с ключом `"user_email"`), а также поддерживает кастомный метод `Validate() error`.
5. **Динамический JSON без структур**: безопасная навигация по вложенным объектам и массивам без кастинга через `map[string]any` (`node.Get("company.employees.0.name").String()`).

---

## Установка

```bash
go get github.com/s7ukk/stuck-json
```

Для сборки и запуска тестов:
```bash
make test
# или
go test -v ./...
```

---

## Примеры использования

### 1. HTTP Handler (Биндинг + Валидация + Дефолты в 1 строку)

```go
package main

import (
	"net/http"
	stuckjson "github.com/s7ukk/stuck-json"
)

type RegisterRequest struct {
	Username string   `json:"username" validate:"required,min=3,max=30"`
	Email    string   `json:"email" validate:"required,email"`
	Age      int      `json:"age" validate:"min=18,max=120" default:"18"`
	Role     string   `json:"role" validate:"oneof=admin|user|manager" default:"user"`
	Tags     []string `json:"tags" default:"newbie,registered"`
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	// В одной функции: проверка размера body, JSON-декодирование,
	// заполнение дефолтов и валидация всех полей!
	req, err := stuckjson.BindAndValidate[RegisterRequest](r)
	if err != nil {
		stuckjson.WriteError(w, http.StatusBadRequest, err)
		return
	}

	// Отправка JSON ответа:
	_ = stuckjson.Write(w, http.StatusCreated, map[string]any{
		"message": "User successfully registered",
		"user":    req,
	})
}
```

Если клиент передаст:
```json
{
  "username": "al",
  "email": "invalid-email"
}
```
`stuckjson.WriteError` автоматически вернет клиенту понятный JSON со статусом 400:
```json
{
  "error": "validation_failed",
  "message": "Входные данные содержат ошибки",
  "details": {
    "age": "значение поля 'age' должно быть не меньше 18",
    "email": "поле 'email' должно содержать корректный email адрес",
    "username": "минимальная длина поля 'username' — 3 симв."
  }
}
```

---

### 2. Подгрузка значений по умолчанию (`default:"..."`)

Если в JSON поле не передано или имеет нулевое значение, `stuck-json` автоматически заполняет его:

| Тег | Поведение |
| :--- | :--- |
| `default:"active"` | Устанавливает строку `"active"` |
| `default:"21"` | Устанавливает число `21` |
| `default:"true"` | Устанавливает булев флаг `true` |
| `default:"5s"` | Парсит в `time.Duration` |
| `default:"a,b,c"` | Заполняет слайс строк `[]string{"a", "b", "c"}` |
| `default:"now"` | Текущее время `time.Now().UTC()` |
| `default:"uuid"` | Генерирует уникальный случайный UUIDv4 |

Пример:
```go
type Session struct {
    ID     string `json:"id" default:"uuid"`
    Status string `json:"status" default:"pending"`
}

sess, _ := stuckjson.DecodeAndValidate[Session]([]byte(`{}`))
// sess.ID -> "4e15b2e9-89a3-4cc0-90c7-2c9745dafae7"
// sess.Status -> "pending"
```

---

### 3. Правила валидации (`validate:"..."`)

- `required` — поле обязательно и не должно быть нулевым.
- `min=N` — мин. длина строки, мин. значение числа, мин. элементов в слайсе.
- `max=N` — макс. длина строки, макс. значение числа, макс. элементов в слайсе.
- `email` — строгая проверка корректности email адреса.
- `url` — проверка валидности URL со схемой `http://` или `https://`.
- `uuid` — проверка формата UUIDv4.
- `oneof=val1|val2|val3` — значение должно быть одним из указанных.
- `alphanumeric` — только буквы и цифры.
- `json` — строка должна быть валидным JSON.

---

### 4. Динамический JSON без создания структур (`stuckjson.Parse`)

Если структура JSON заранее неизвестна или нужно быстро достать одно поле из глубины:

```go
raw := []byte(`{
    "store": {
        "book": [
            { "title": "The Go Programming Language", "price": 45.5 }
        ]
    }
}`)

node, _ := stuckjson.Parse(raw)

title := node.Get("store.book.0.title").String() // "The Go Programming Language"
price := node.Get("store.book.0.price").Float64() // 45.5
```

---

### 5. Дженерик-утилиты быстрого декодирования

```go
// Чтение из io.Reader напрямую в тип T
user, err := stuckjson.Decode[User](reader)

// Чтение из []byte
user, err := stuckjson.DecodeBytes[User](bytes)

// Чтение из string
user, err := stuckjson.DecodeString[User](str)

// Преобразование в строку
str, err := stuckjson.ToString(user)

// Красивый отформатированный вывод (Pretty)
pretty, err := stuckjson.Pretty(user)
```

---

## Лицензия

MIT License (c) 2026 stuck-json contributors.
