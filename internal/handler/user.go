package handler

import (
	"fmt"

	"github.com/rom6n/FiberWebApp/internal/database"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/gofiber/fiber/v2"
)

func Register(db *mongo.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		name, nickname, password := c.Query("name"), c.Query("nickname"), c.Query("password")
		if name == "" || nickname == "" || password == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Name, nickname, and password are required")
		}

		err := database.AddUser(db, name, nickname, password)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf("Failed to add user: %v", err))
		}

		return c.SendString(fmt.Sprintf("User created:\nName: %v\nNickname: %v", name, nickname))
	}
}

func Profile(c *fiber.Ctx) error {
	nickname := c.Query("nickname")
	if nickname == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Nickname is required")
	}
	fmt.Println(nickname)
	return nil
}
