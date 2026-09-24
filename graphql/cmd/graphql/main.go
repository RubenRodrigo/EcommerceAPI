package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/rasadov/EcommerceAPI/graphql/config"
	"github.com/rasadov/EcommerceAPI/graphql/graph"
	"github.com/rasadov/EcommerceAPI/pkg/middleware"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	graphServer, err := graph.NewGraphQLServer(config.AccountUrl, config.ProductUrl, config.OrderUrl, config.PaymentUrl, config.RecommenderUrl)
	if err != nil {
		return err
	}
	defer graphServer.Close()

	srv := handler.New(graphServer.ToExecutableSchema())
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})
	srv.SetErrorPresenter(graph.ErrorPresenter)

	engine := gin.Default()

	engine.Use(middleware.GinContextToContextMiddleware())

	engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "It works",
		})
	})
	engine.POST("/graphql",
		middleware.AuthorizeJWT(),
		func(c *gin.Context) {
			ctx := graphServer.WithLoaders(c.Request.Context())
			c.Request = c.Request.WithContext(ctx)
			c.Next()
		},
		gin.WrapH(srv),
	)
	engine.GET("/playground", gin.WrapH(playground.Handler("Playground", "/graphql")))

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: engine,
	}
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- httpServer.ListenAndServe()
	}()

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-shutdownSignal.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	}
}
