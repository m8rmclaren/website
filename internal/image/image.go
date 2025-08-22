package image

import "github.com/labstack/echo/v4"

type optimizer struct {
}

func NewImageOptimizer() *optimizer {
	return &optimizer{}
}

func (o *optimizer) Handler() echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Logger().Printf("Serving GET %s, [ip %s]", c.Path(), c.RealIP())
		return nil
	}
}
