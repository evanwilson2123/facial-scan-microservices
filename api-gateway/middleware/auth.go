package middleware

import (
	"log"
	"strings"

	"api-gateway/utils"

	"github.com/gofiber/fiber/v2"
)

func AuthRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			log.Println("No header")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"missing or invalid token"})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Println("bad header")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"missing or invalid token"})
		}

		tokenString := parts[1]

		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			log.Println("Invalid or expired")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"invalid or expired token"})
		}

		c.Locals("user_id", claims.UID)
		c.Set("X-User-ID", claims.UID)
		return c.Next()
	}
}