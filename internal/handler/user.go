package handler

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/rom6n/FiberWebApp/internal/database"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/gofiber/fiber/v2"
)

func Register(mongoClient *mongo.Client, redisClient *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()

		name, nickname, password := c.Query("name"), c.Query("nickname"), c.Query("password")
		if name == "" || nickname == "" || password == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Name, nickname, and password are required")
		}

		err := database.AddUser(ctx, mongoClient, redisClient, name, nickname, password)
		if err != nil {
			if err == context.DeadlineExceeded {
				return c.Status(fiber.StatusServiceUnavailable).SendString("Error. Try again")
			}
			return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf("Failed to add user: %v", err))
		}

		return c.SendString(fmt.Sprintf("User created:\nName: %v\nNickname: %v", name, nickname))
	}
}

func Login(mongoClient *mongo.Client, redisClient *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()

		nickname, password := c.Query("nickname"), c.Query("password")
		if nickname == "" || password == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Nickname and password are required")
		}

		user, findErr := database.FindUserInCacheByNickname(ctx, mongoClient, redisClient, nickname)

		if findErr != nil {
			if findErr == context.DeadlineExceeded {
				return c.Status(fiber.StatusServiceUnavailable).SendString("Error. Try again")
			}
			return c.Status(fiber.StatusNotFound).SendString("Account does not exists")
		}

		isRight, verifyErr := database.VerifyHash(password, user.Password)
		if verifyErr != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Wrong password. try again")
		}

		if !isRight {
			return c.Status(fiber.StatusBadRequest).SendString("Wrong password, try again")
		}
		c.Cookie(&fiber.Cookie{
			Name:     "Authorization",
			Value:    fmt.Sprint(user.Id),
			SameSite: "Lax",
		})
		return c.SendString("Successfully login")
	}
}

func Profile(mongoClient *mongo.Client, redisClient *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()

		nickname := c.Query("nickname")
		if nickname == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Nickname is required")
		}

		authToken := c.Cookies("Authorization")

		user, err := database.FindUserInCacheByNickname(ctx, mongoClient, redisClient, nickname)
		if err != nil {
			if err == context.DeadlineExceeded {
				return c.Status(fiber.StatusServiceUnavailable).SendString("Error. Try again")
			}
			return c.Status(fiber.StatusNotFound).SendString("Account does not exists")
		}

		var ownerPrefix string
		if authToken == fmt.Sprint(user.Id) {
			ownerPrefix = "Your "
		}

		return c.Status(fiber.StatusFound).SendString(fmt.Sprintf(
			ownerPrefix+"Profile:\nID: %v\nName: %v\nNickname: %v\nPassword: %v", user.Id, user.Name, user.Nickname, user.Password,
		))
	}
}
