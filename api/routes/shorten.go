package routes

import (
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/ronak-pal1/url-shortener-go/database"
	"github.com/ronak-pal1/url-shortener-go/helpers"

	"github.com/asaskevich/govalidator"
	"github.com/gofiber/fiber/v2"
)

type request struct {
	URL         string        `json:"url"`
	CustomShort string        `json:"short"`
	Expiry      time.Duration `json:"expiry"`
}

type response struct {
	URL             string        `json:"url"`
	CustomShort     string        `json:"short"`
	Expiry          time.Duration `json:"expiry"`
	XRateRemaining  int           `json:"rate_limit"`
	XRateLimitReset time.Duration `json:"rate_limit_reset"`
}

func ShortenURL(c *fiber.Ctx) error {

	body := new(request)

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	//check rate limiting
	r2 := database.CreateClient(1)

	val, err := r2.Get(database.Ctx, c.IP()).Result()

	if err == redis.Nil {
		_ = r2.Set(database.Ctx, c.IP(), os.Getenv("API_QUOTA"), 30*60*time.Second).Err()
	} else {
		valInt, _ := strconv.Atoi(val)

		if valInt <= 0 {
			limit, _ := r2.TTL(database.Ctx, c.IP()).Result()

			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":            "Rate limit exceeded",
				"rate_limit_reset": limit / time.Nanosecond / time.Minute,
			})

		}
	}

	// check if it's a actual url
	if !govalidator.IsURL(body.URL) {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "Invalid URL"})
	}

	// check domain error
	if !helpers.RemoveDomainError(body.URL) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "domain error"})
	}

	// enforce http
	body.URL = helpers.EnforceHTTP(body.URL)

	r := database.CreateClient(0)

	defer r.Close()

	var id string

	if body.CustomShort == "" {
		id = uuid.New().String()[:6]
	} else {
		id = body.CustomShort
	}

	r_val, _ := r.Get(database.Ctx, id).Result()

	if r_val != "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "custom short is already in use",
		})
	}

	if body.Expiry == 0 {
		body.Expiry = 24
	}

	err = r.Set(database.Ctx, id, body.URL, body.Expiry*3600*time.Second).Err()

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Unable to connect to the server",
		})
	}

	r2.Decr(database.Ctx, c.IP())

	resp := response{
		URL:             body.URL,
		CustomShort:     "",
		Expiry:          body.Expiry,
		XRateRemaining:  10,
		XRateLimitReset: 30,
	}

	rate_val, _ := r2.Get(database.Ctx, c.IP()).Result()

	resp.XRateRemaining, _ = strconv.Atoi(rate_val)

	ttl, _ := r2.TTL(database.Ctx, c.IP()).Result()

	resp.XRateLimitReset = ttl / time.Nanosecond / time.Minute

	resp.CustomShort = os.Getenv("DOMAIN") + "/" + id

	return c.Status(fiber.StatusOK).JSON(resp)
}
