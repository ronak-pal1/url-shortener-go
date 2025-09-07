package routes

import (
	"github.com/asaskevich/govalidator"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/ronak-pal1/url-shortener-go/database"
)


func ResolveURL(c *fiber.Ctx) error {

	url := c.Params("url")

	if !govalidator.IsURL(url) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"Please provide a valid url to resolve"})
	}
	
	r := database.CreateClient(0)

	defer r.Close()

	value, err := r.Get(database.Ctx, url).Result()


	if err == redis.Nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "short not found"})
	}else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error":"Can't connect to the database"})
	}

	rInr := database.CreateClient(1);

	defer rInr.Close()

	_ = rInr.Incr(database.Ctx,"counter")
	
	return c.Redirect(value, 301);	

}