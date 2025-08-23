package api

import "github.com/labstack/echo/v4"

type APIError struct {
	Error string `json:"error"`
}

func RespondError(c echo.Context, code int, err error) error {
	return c.JSON(code, APIError{Error: err.Error()})
}
