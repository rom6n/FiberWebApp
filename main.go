package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rom6n/FiberWebApp/internal/database"
	"github.com/rom6n/FiberWebApp/internal/handler"

	"github.com/gofiber/fiber/v2"

	"github.com/goccy/go-json"

	//fiberlog "github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
)

func main() {
	mongoClient := database.NewMongoClient()
	redisClient := database.NewRedisClient()
	ctx := context.Background()

	signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	defer func() {
		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Fatalf("Failed to disconnect from MongoDB: %v", err)
		}
	}()

	log.SetPrefix("⚡ Fiber: ")
	log.SetFlags(0)
	log.Println("is starting")

	app := fiber.New(fiber.Config{
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	})

	app.Use(logger.New(logger.Config{
		DisableColors: false,
	}))

	app.Use(csrf.New(csrf.ConfigDefault))

	app.Use(compress.New(compress.Config{
		Level: 1,
	}))

	app.Get("/metrics", monitor.New(monitor.Config{
		Title:   "Fiber Web App",
		Refresh: 1 * time.Second,
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("You on '/' page")
	})

	app.Use(limiter.New(limiter.Config{
		Max:        20,
		Expiration: 10 * time.Second,
	}))

	app.Get("/ip", func(c *fiber.Ctx) error {
		return c.SendString(c.IP())
	})

	app.Get("/json", func(c *fiber.Ctx) error {
		user := struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		}{
			ID:    123,
			Name:  "John Doe",
			Email: "john@example.com",
		}
		return c.JSON(user)
	})

	app.Get("/query", func(c *fiber.Ctx) error {
		return c.SendString(c.Query("id"))
	})

	app.Get("/d", func(c *fiber.Ctx) error {
		return c.Download("./static/3d-art-dark-3840x2160-12034.jpg", "abstract-wallpaper.jpg")
	})

	app.Get("/register", handler.Register(mongoClient, redisClient))

	app.Get("/profile", handler.Profile(mongoClient, redisClient))

	app.Get("/login", handler.Login(mongoClient, redisClient))

	app.Get("/:name", func(c *fiber.Ctx) error {
		msg := fmt.Sprintf("<h1>Hello, %v dev!</h1>", c.Params("name"))
		return c.Status(fiber.StatusOK).Format(msg)
	})
	//---------------------------------------------------------------

	go func() {
		<-signalCtx.Done()
		shutdownCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		fmt.Println("🌳 Graceful Shutdown 🌳")
		defer stop()
		app.ShutdownWithContext(shutdownCtx)
	}()

	app.Listen(":3000")

}
