# Розробка REST-сервісу на базі фреймворку Gin та СУБД PostgreSQL

**Практична робота №5** · Incident Tracker API

---

## Мета

- Опанувати принципи розробки вебсервісів на базі фреймворку Gin (`gin-gonic/gin`) та взаємодію з контекстом запиту `gin.Context`.
- Навчитися застосовувати декларативну валідацію вхідних DTO за допомогою тегів `binding` (`go-playground/validator`).
- Опанувати підключення до PostgreSQL, проєктування моделей даних та виконання операцій через ORM-бібліотеку GORM.
- Засвоїти хешування паролів алгоритмом bcrypt та випуск підписаних JWT-токенів (`golang-jwt/jwt/v5`).
- Навчитися захищати приватні маршрути кастомним JWT Middleware та генерувати інтерактивну документацію API за допомогою Swagger UI (`swaggo/swag`).

---

## 2. Предметна область: облік технічних інцидентів

Incident Tracker API — сервіс обліку технічних інцидентів та аварій. Інженери реєструються та авторизуються, отримуючи JWT-токен. Перегляд інцидентів публічний, а створення, зміна статусу й видалення доступні лише з валідним токеном. Інцидент прив'язується до користувача, який його створив.

### Сутності

**Incident**

| Поле | Тип | Опис |
|---|---|---|
| `id` | uint | первинний ключ |
| `title` | string (до 100) | заголовок |
| `description` | string | докладний опис |
| `priority` | string | `low`, `medium`, `high`, `critical` |
| `status` | string | `open` (за замовчуванням), `in_progress`, `resolved`, `closed` |
| `user_id` | uint | ID інженера, який створив інцидент |
| `created_at` | time | час створення |

**User**

| Поле | Тип | Опис |
|---|---|---|
| `id` | uint | первинний ключ |
| `email` | string | унікальний індекс |
| `password_hash` | string | bcrypt-геш, приховано від JSON (`json:"-"`) |
| `created_at` | time | час реєстрації |

### Ендпоінти

| Метод | Маршрут | Доступ | Успіх | Помилки |
|---|---|---|---|---|
| POST | `/api/v1/auth/register` | публічний | 201 | 400 |
| POST | `/api/v1/auth/login` | публічний | 200 + `token` | 400, 401 |
| GET | `/api/v1/incidents` | публічний | 200 | 500 |
| GET | `/api/v1/incidents/:id` | публічний | 200 | 404 |
| POST | `/api/v1/incidents` | **JWT** | 201 | 400, 401 |
| PATCH | `/api/v1/incidents/:id/status` | **JWT** | 200 | 400, 401, 404 |
| DELETE | `/api/v1/incidents/:id` | **JWT** | 204 | 401, 404 |
| GET | `/swagger/*any` | публічний | Swagger UI | — |

---

## 3. Технології

- Go 1.22+
- Gin (`github.com/gin-gonic/gin`)
- GORM + драйвер PostgreSQL (`gorm.io/gorm`, `gorm.io/driver/postgres`)
- PostgreSQL (Docker, образ `postgres:alpine`)
- bcrypt (`golang.org/x/crypto/bcrypt`)
- JWT (`github.com/golang-jwt/jwt/v5`)
- Swagger (`swaggo/swag`, `swaggo/gin-swagger`, `swaggo/files`)

---

## 4. Структура проєкту

```text
pr5-client-server/
├── database/     # ініціалізація підключення GORM до PostgreSQL
├── dto/          # DTO з тегами валідації (binding)
├── model/        # моделі Incident та User
├── docs/         # згенерована Swagger-документація (swag init)
├── main.go       # маршрути, обробники, AuthMiddleware
├── go.mod
└── go.sum
```

---

## 5. Запуск проєкту

### 1. База даних (Docker)

```bash
docker run -d --name postgres-dev -e POSTGRES_PASSWORD=postgres -p 5432:5432 postgres:alpine
```

### 2. Застосунок

```bash
git clone https://github.com/trrt54999/client-server-rest-server-gin-and-postgresql
cd pr5-client-server
go mod download
go run .
```

Сервіс запускається на `http://localhost:8080`. Таблиці `users` та `incidents` створюються автоматично через `AutoMigrate`.

### 3. Swagger UI

Після запуску документація доступна за адресою: `http://localhost:8080/swagger/index.html`.

---

## 6. Хід роботи

### Рівень 1. Gin, PostgreSQL та валідація DTO

**1. Підключення до PostgreSQL** (`database`). У разі помилки підключення застосунок зупиняється.

```go
func InitDB() *gorm.DB {
	dsn := "host=localhost user=postgres password=postgres dbname=postgres port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	return db
}
```

<img width="2535" height="1439" alt="image" src="https://github.com/user-attachments/assets/282d40ff-d033-45c4-b529-a7ca6411b9b8" />

**2. Модель Incident** (`model`):

```go
type Incident struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"size:100; not null" json:"title"`
	Description string    `gorm:"not null" json:"description"`
	Priority    string    `gorm:"size:20;not null" json:"priority"`
	Status      string    `gorm:"size:20;default:'open'" json:"status"`
	UserID      uint      `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
}
```

Авто-міграція виконується в `main()`: `db.AutoMigrate(&model.Incident{}, &model.User{})`.

<img width="1094" height="258" alt="image" src="https://github.com/user-attachments/assets/1a917365-01c5-4532-b981-9aa4d9f0bd29" />


<img width="1079" height="434" alt="image" src="https://github.com/user-attachments/assets/cf70b75e-ddbb-4433-b9e5-c364741390c1" />

**3. DTO з декларативною валідацією** (`dto`):

```go
type CreateIncidentDto struct {
	Title       string `json:"title" binding:"required,min=3,max=100"`
	Description string `json:"description" binding:"required"`
	Priority    string `json:"priority" binding:"required,oneof=low medium high critical"`
}
```

**4. Обробники.** Тіло запиту десеріалізується та валідується одним викликом `c.ShouldBindJSON`. У разі невідповідності сервер повертає `400 Bad Request`:

```go
var input dto.CreateIncidentDto
if err := c.ShouldBindJSON(&input); err != nil {
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	return
}
```

`GET /api/v1/incidents` вичитує всі записи через `db.Find(&incidents)` і повертає `200 OK`.

---

### Рівень 2. Повний CRUD та автентифікація

**1. Решта CRUD-операцій над інцидентами:**

- `GET /api/v1/incidents/:id` — пошук через `db.First(&incident, id)`, `404`, якщо запис не знайдено.
- `PATCH /api/v1/incidents/:id/status` — оновлення статусу з валідацією нового значення.
- `DELETE /api/v1/incidents/:id` — видалення запису, відповідь `204 No Content`.

```go
type UpdateIncidentStatusDto struct {
	Status string `json:"status" binding:"required,oneof=open in_progress resolved closed"`
}
```

**2. Модель користувача.** Геш пароля не потрапляє в JSON завдяки тегу `json:"-"`, а унікальність email забезпечує `uniqueIndex`.

```go
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
```

**3. Реєстрація** (`POST /api/v1/auth/register`). Пароль гешується bcrypt, у базі зберігається лише геш:

```go
type RegisterUserDto struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}
```

```go
hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
// ...
user := model.User{
	Email:        input.Email,
	PasswordHash: string(hashedPassword),
}
```

Якщо користувач із таким email уже існує, сервіс повертає `400` з повідомленням `User with this email already exists!`.

**4. Логін** (`POST /api/v1/auth/login`). Користувач шукається за email, пароль перевіряється через `bcrypt.CompareHashAndPassword`. При невідповідності (невідомий email або хибний пароль) повертається однакове `401`, щоб не розкривати, чи існує обліковий запис. При успіху генерується JWT (HS256) із терміном дії 24 години:

```go
claims := jwt.MapClaims{
	"user_id": user.ID,
	"email":   user.Email,
	"exp":     time.Now().Add(time.Hour * 24).Unix(),
}

token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
tokenString, err := token.SignedString([]byte(jwtSecret))
```

Відповідь: `{"token": "<jwt_string>"}`.

---

### Рівень 3. JWT Middleware та Swagger UI

**1. AuthMiddleware.** Витягує заголовок `Authorization`, перевіряє префікс `Bearer `, верифікує підпис токена та зберігає `userID` у контексті. У разі будь-якої невідповідності ланцюжок обробників негайно переривається через `AbortWithStatusJSON`:

```go
func AuthMiddleware(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, http.ErrAbortHandler
			}
			return []byte(secretKey), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
			return
		}

		userIdFloat, ok := claims["user_id"].(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
			return
		}

		c.Set("userID", uint(userIdFloat))
		c.Next()
	}
}
```

**2. Захист приватних маршрутів.** Публічні маршрути (`register`, `login`, `GET /incidents`, `GET /incidents/:id`) лежать у групі `v1`, а модифікуючі операції винесені в підгрупу з middleware:

```go
secured := v1.Group("")
secured.Use(AuthMiddleware(jwtSecret))
{
	secured.POST("/incidents", ...)
	secured.PATCH("/incidents/:id/status", ...)
	secured.DELETE("/incidents/:id", ...)
}
```

**3. Прив'язка інциденту до користувача.** При створенні `userID` береться з контексту, який заповнив middleware:

```go
userID, _ := c.Get("userID")
// ...
incident := model.Incident{
	Title:       input.Title,
	Description: input.Description,
	Priority:    input.Priority,
	Status:      "open",
	UserID:      userID.(uint),
}
```

**4. Swagger UI.** Документація генерується утилітою `swag` із декларативних анотацій над `main()` та обробниками (загальна інформація про API, схема безпеки `BearerAuth`, `@Summary`, `@Tags`, `@Param`, `@Success`, `@Failure`, `@Security BearerAuth`). Маршрут підключено так:

```go
r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
```

Для авторизації в Swagger UI натискається кнопка **Authorize**, у поле вводиться значення `Bearer <token>` (токен береться з відповіді `/auth/login`).

---

## 7. Демонстрація тестування API

Тестування виконувалося через Swagger UI. Значення `id` інциденту залежить від вмісту таблиці, у прикладах використано `1`.

### Запуск сервера

<img width="2511" height="1405" alt="image" src="https://github.com/user-attachments/assets/6b8447ee-530b-4f17-9bad-05c56a4dcf58" />

---

### 1. Реєстрація користувача

- **Request:** `POST http://localhost:8080/api/v1/auth/register`
- **Headers:** `Content-Type: application/json`
- **Body:**

```json
{
  "email": "veresshagin999@company.internal",
  "password": "neimovirno15"
}
```

- **Очікуваний результат:** `201 Created`, у відповіді `id`, `email`, `created_at`. Поле `password_hash` відсутнє.

<img width="1573" height="1289" alt="image" src="https://github.com/user-attachments/assets/073324a3-ecfc-4f1d-8653-36aed06b2e71" />


---

### 2. Реєстрація з коротким паролем

- **Request:** `POST http://localhost:8080/api/v1/auth/register`
- **Body:** `{"email": "veresshagin9999@company.internal", "password": "123"}`
- **Очікуваний результат:** `400 Bad Request`, помилка валідації поля `Password` (правило `min`).

<img width="1571" height="1010" alt="image" src="https://github.com/user-attachments/assets/b2c46559-7436-4d42-a52e-cab11531ad28" />

---

### 3. Повторна реєстрація з тим самим email

- **Request:** `POST http://localhost:8080/api/v1/auth/register` (тіло з п. 1)
- **Очікуваний результат:** `400 Bad Request`, `{"error": "User with this email already exists!"}`.

<img width="1445" height="1266" alt="image" src="https://github.com/user-attachments/assets/42b60809-ea40-4f56-94d3-03afd38193ee" />

---

### 4. Логін з хибним паролем

- **Request:** `POST http://localhost:8080/api/v1/auth/login`
- **Body:** `{"email": "veresshagin99@company.internal", "password": "12123123"}`
- **Очікуваний результат:** `401 Unauthorized`.

<img width="1476" height="1270" alt="image" src="https://github.com/user-attachments/assets/6d32a66d-3aed-4e1f-a3a9-15cd5d4cf3d9" />


---

### 5. Успішний логін та отримання JWT

- **Request:** `POST http://localhost:8080/api/v1/auth/login`
- **Body:**

```json
{
  "email": "veresshagin99@company.internal",
  "password": "neimovirno15"
}
```

- **Очікуваний результат:** `200 OK`, `{"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InZlcmVzc2hhZ2luOTlAY29tcGFueS5pbnRlcm5hbCIsImV4cCI6MTc5Mjk2NDUxOSwidXNlcl9pZCI6N30.bTAoJOIztM9-_AJmsWppKvtwyxlHFkGZtceDZDgu6GY"}`. Цей токен використовується в наступних запитах.

<img width="1451" height="1302" alt="image" src="https://github.com/user-attachments/assets/d61b8111-e3f3-4075-8baa-458e4afb9224" />

---

### 6. Створення інциденту без токена

- **Request:** `POST http://localhost:8080/api/v1/incidents`
- **Body:** `{"title": "bla bla bla", "description": "bla bla bla", "priority": "critical"}`
- **Очікуваний результат:** `401 Unauthorized`, `{"error": "Unauthorized access"}`.

<img width="1320" height="1171" alt="image" src="https://github.com/user-attachments/assets/677682e6-26bd-45c1-8a22-4991debf3dc2" />

---

### 7. Невалідний пріоритет (валідація DTO)

- **Request:** `POST http://localhost:8080/api/v1/incidents`
- **Headers:** `Content-Type: application/json`, `Authorization: Bearer <TOKEN>`
- **Body:**

```json
{
  "title": "bla bla bla",
  "description": "bla bla bla",
  "priority": "idk"
}
```

- **Очікуваний результат:** `400 Bad Request`, повідомлення про помилку валідації поля `Priority` (правило `oneof`).

<img width="1319" height="1279" alt="image" src="https://github.com/user-attachments/assets/e7b8ecd7-9f94-422d-9291-561f24191bab" />

---

### 8. Створення інциденту з валідним токеном

- **Request:** `POST http://localhost:8080/api/v1/incidents`
- **Headers:** `Content-Type: application/json`, `Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InZlcmVzc2hhZ2luOTlAY29tcGFueS5pbnRlcm5hbCIsImV4cCI6MTc5Mjk2NDUxOSwidXNlcl9pZCI6N30.bTAoJOIztM9-_AJmsWppKvtwyxlHFkGZtceDZDgu6GY`
- **Body:**

```json
{
  "title": "Overheating on Gateway-01",
  "description": "Core temperature reached 85C",
  "priority": "critical"
}
```

- **Очікуваний результат:** `201 Created`, об'єкт із `status: "open"` та `user_id` авторизованого користувача.

<img width="1410" height="1274" alt="image" src="https://github.com/user-attachments/assets/d0d1ecbc-39b2-4d92-bba7-b130c17a4fdb" />

---

### 9. Отримання списку інцидентів

- **Request:** `GET http://localhost:8080/api/v1/incidents`
- **Очікуваний результат:** `200 OK`, масив інцидентів (токен не потрібен).

<img width="1320" height="962" alt="image" src="https://github.com/user-attachments/assets/ded5896e-bcaf-49b3-be83-197e59ae70d8" />

---

### 10. Отримання інциденту за ID

- **Request:** `GET http://localhost:8080/api/v1/incidents/1`
- **Очікуваний результат:** `200 OK` та об'єкт інциденту.

<img width="1493" height="1097" alt="image" src="https://github.com/user-attachments/assets/cb39ad43-09f5-4a95-be1b-5fa0ecc311db" />

---

### 11. Обробка помилки 404

- **Request:** `GET http://localhost:8080/api/v1/incidents/1312`
- **Очікуваний результат:** `404 Not Found`, `{"error": "Incident not found!"}`.

<img width="1503" height="949" alt="image" src="https://github.com/user-attachments/assets/359cb737-ace5-4e18-90c6-ed8d66bb911b" />

---

### 12. Зміна статусу інциденту

- **Request:** `PATCH http://localhost:8080/api/v1/incidents/7/status`
- **Headers:** `Content-Type: application/json`, `Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InZlcmVzc2hhZ2luOTlAY29tcGFueS5pbnRlcm5hbCIsImV4cCI6MTc5Mjk2NDUxOSwidXNlcl9pZCI6N30.bTAoJOIztM9-_AJmsWppKvtwyxlHFkGZtceDZDgu6GY`
- **Body:** `{"status": "in_progress"}`
- **Очікуваний результат:** `200 OK`, у відповіді інцидент зі статусом `in_progress`.

<img width="1359" height="1206" alt="image" src="https://github.com/user-attachments/assets/48ca8ebc-d9c1-4894-b1a7-b7783853b99e" />

---

### 13. Невалідний статус

- **Request:** `PATCH http://localhost:8080/api/v1/incidents/1/status`
- **Headers:** `Content-Type: application/json`, `Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InZlcmVzc2hhZ2luOTlAY29tcGFueS5pbnRlcm5hbCIsImV4cCI6MTc5Mjk2NDUxOSwidXNlcl9pZCI6N30.bTAoJOIztM9-_AJmsWppKvtwyxlHFkGZtceDZDgu6GY`
- **Body:** `{"status": "done"}`
- **Очікуваний результат:** `400 Bad Request`, помилка валідації поля `Status` (правило `oneof`).

<img width="1620" height="1302" alt="image" src="https://github.com/user-attachments/assets/1cb78948-75a1-4712-b784-21015eeb51e5" />

---

### 14. Видалення без токена

- **Request:** `DELETE http://localhost:8080/api/v1/incidents/6`
- **Очікуваний результат:** `401 Unauthorized`, `{"error": "Unauthorized access"}`.

<img width="1637" height="983" alt="image" src="https://github.com/user-attachments/assets/a53b9da9-10b6-4532-b583-2c1f8e0e8f89" />

---

### 15. Видалення з валідним токеном

- **Request:** `DELETE http://localhost:8080/api/v1/incidents/6`
- **Headers:** `Authorization: Bearer <TOKEN>`
- **Очікуваний результат:** `204 No Content` без тіла відповіді. Повторний запит на цей `id` повертає `404`.

<img width="1607" height="1268" alt="image" src="https://github.com/user-attachments/assets/fc62f66a-14d3-4b4a-b066-9095eb1d40fc" />

---

### 16. Swagger UI

- **URL:** `http://localhost:8080/swagger/index.html`

<img width="1506" height="1285" alt="image" src="https://github.com/user-attachments/assets/584b2dc7-3e35-4cfe-ad99-b2e486bb9fd6" />

---

## 8. Висновки

Було розроблено RESTful-сервіс Incident Tracker API на фреймворку Gin з використанням PostgreSQL.

Зокрема:

1. Організовано маршрутизацію через групи `/api/v1`, а обробники працюють із запитом через `gin.Context`.
2. Вхідні дані перевіряються декларативно за допомогою тегів `binding` у DTO (`required`, `min`, `max`, `email`, `oneof`), помилки валідації повертаються зі статусом `400`.
3. Налаштовано підключення до PostgreSQL у Docker, моделі `Incident` та `User` синхронізуються зі схемою через `AutoMigrate` і GORM.
4. Реалізовано повний CRUD над інцидентами з коректними статусами відповідей (`201`, `200`, `204`, `404`).
5. Паролі зберігаються у вигляді bcrypt-гешів, а при успішному логіні видається JWT із терміном дії 24 години.
6. Написано кастомний `AuthMiddleware`, який захищає створення, зміну та видалення інцидентів, а інцидент прив'язується до `userID` авторизованого користувача.
7. Підключено Swagger UI для інтерактивної документації та тестування API, включно з авторизацією через `BearerAuth`.
