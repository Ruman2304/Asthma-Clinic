package controller

import (
	"asthma-clinic/configuration"
	"asthma-clinic/middleware"
	"asthma-clinic/models"
	"context"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

func Register(c *fiber.Ctx) error {
	var body models.RegisterRequest

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if body.Name == "" || body.Email == "" || body.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Name, email and password are required"})
	}

	if len(body.Password) < 6 {
		return c.Status(400).JSON(fiber.Map{"error": "Password must be at least 6 characters"})
	}

	// Check email uniqueness
	var exists bool
	err := configuration.DB.QueryRow(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)`, body.Email,
	).Scan(&exists)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Database error"})
	}
	if exists {
		return c.Status(409).JSON(fiber.Map{"error": "Email already registered"})
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to hash password"})
	}

	var id int
	err = configuration.DB.QueryRow(context.Background(),
		`INSERT INTO users (name, email, password) VALUES ($1, $2, $3) RETURNING id`,
		body.Name, body.Email, string(hash),
	).Scan(&id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	token, err := middleware.GenerateToken(id, body.Email, body.Name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate token"})
	}

	return c.Status(201).JSON(models.AuthResponse{
		Token: token,
		User:  models.User{ID: id, Name: body.Name, Email: body.Email},
	})
}

func Login(c *fiber.Ctx) error {
	var body models.LoginRequest

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var user models.User
	err := configuration.DB.QueryRow(context.Background(),
		`SELECT id, name, email, password, created_at FROM users WHERE email=$1`,
		body.Email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.CreatedAt)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid email or password"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password)); err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid email or password"})
	}

	token, err := middleware.GenerateToken(user.ID, user.Email, user.Name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate token"})
	}

	user.Password = ""
	return c.JSON(models.AuthResponse{Token: token, User: user})
}

func GetMe(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	var user models.User
	err := configuration.DB.QueryRow(context.Background(),
		`SELECT id, name, email, created_at FROM users WHERE id=$1`, userID,
	).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}
	return c.JSON(user)
}
