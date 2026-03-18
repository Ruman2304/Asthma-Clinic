package controller

import (
	"asthma-clinic/configuration"
	"asthma-clinic/models"
	"context"

	"github.com/gofiber/fiber/v2"
)

func GetAllUsers(c *fiber.Ctx) error {
	rows, err := configuration.DB.Query(
		context.Background(),
		`SELECT id, name, email, created_at FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		users = append(users, u)
	}

	if users == nil {
		users = []models.User{}
	}
	return c.JSON(users)
}

func CreateUser(c *fiber.Ctx) error {
	var body models.User
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var id int
	err := configuration.DB.QueryRow(
		context.Background(),
		`INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id`,
		body.Name, body.Email,
	).Scan(&id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(fiber.Map{"message": "User created successfully", "id": id})
}
