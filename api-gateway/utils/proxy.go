package utils

import (
	"bytes"
	"io"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func Proxy(c *fiber.Ctx, target string) error {
    targetURL := target + c.OriginalURL()
    log.Printf("Proxying request to %s\n", targetURL)
    log.Printf("Request method: %s\n", c.Method())
    log.Printf("Request body: %s\n", string(c.Body()))

    // Create a new HTTP request
    req, err := http.NewRequest(c.Method(), targetURL, bytes.NewReader(c.Body()))
    if err != nil {
        log.Printf("Failed to create request: %v\n", err)
        return err
    }

    // Copy headers from the original request
    c.Request().Header.VisitAll(func(key, value []byte) {
        req.Header.Set(string(key), string(value))
        log.Printf("Copying header: %s: %s\n", key, value)
    })

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        log.Printf("Failed to forward request: %v\n", err)
        return err
    }
    defer resp.Body.Close()

    // Log the response status code
    log.Printf("Response status code: %d\n", resp.StatusCode)

    // Copy headers from the response
    for key, values := range resp.Header {
        for _, value := range values {
            c.Set(key, value)
            log.Printf("Setting response header: %s: %s\n", key, value)
        }
    }

    // Copy status code
    c.Status(resp.StatusCode)

    // Read the response body
    responseBody, err := io.ReadAll(resp.Body)
    if err != nil {
        log.Printf("Failed to read response body: %v\n", err)
        return err
    }
    log.Printf("Response body: %s\n", string(responseBody))

    // Write the response body to the client
    _, err = c.Response().BodyWriter().Write(responseBody)
    if err != nil {
        log.Printf("Failed to write response body: %v\n", err)
        return err
    }

    return nil
}
