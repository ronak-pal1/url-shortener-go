package routes

import "github.com/gofiber/fiber/v2"


func ResolveURL(c *fiber.Ctx) error {
	
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "URL is successfully resolved"})
}