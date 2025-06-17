package main

import "github.com/gin-gonic/gin"

func main() {
	e := gin.Default()

	e.GET("/", func(ctx *gin.Context) {
		ctx.JSON(200)
	})

	e.Run(":8080")

}
