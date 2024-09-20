package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Hello-Storage/hello-storage-proxy/internal/config"
	"github.com/Hello-Storage/hello-storage-proxy/internal/middlewares"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Start(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()

	start := time.Now()

	gin.SetMode(gin.DebugMode)

	// Create new HTTP router engine without standard middleware.
	router := gin.New()

	ProtectedRouter := gin.New()

	router.MaxMultipartMemory = 500 << 20

	protectedCorsConfig := cors.New(cors.Config{
		AllowOrigins: []string{
			//development
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			//production
			//ipfs
			"http://bafybeicm4w4vstqisfm6y67sfuusemftosfhmf4kluzeczaweyynunxele.ipfs.localhost:8080",
			"https://bafybeicm4w4vstqisfm6y67sfuusemftosfhmf4kluzeczaweyynunxele.ipfs.cf-ipfs.com",
			"https://bafybeicm4w4vstqisfm6y67sfuusemftosfhmf4kluzeczaweyynunxele.ipfs.dweb.link",
			"https://bafybeicm4w4vstqisfm6y67sfuusemftosfhmf4kluzeczaweyynunxele.ipfs.infura-ipfs.io",
			"https://ipfs.hello.app",
			"https://www.ipfs.hello.app",
			// normal
			"https://helloapp.eth",
			"https://www.helloapp.eth",
			"https://joinhello.app",
			"https://staging.joinhello.app",
			"https://www.staging.joinhello.app",
			"https://www.joinhello.app",
			"https://joinhello.vercel.app",
			"https://www.joinhello.vercel.app",
			"https://hello.storage",
			"https://www.hello.storage",
			"https://space.hello.app",
			"https://hello.app",
			"https://www.hello.app",
			"https://stats.hello.app",
			"https://www.stats.hello.app",
			"https://www.space.hello.app",
			"https://space.hello.storage",
			"https://www.space.hello.storage",
			"https://space.hello.ws",
			"https://www.space.hello.ws",
		},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Cross-Origin-Opener-Policy",
			"Access-Control-Allow-Origin",
			"Authorization",
			"Api_key",
			"api_key",
			"api-key",
		},
		AllowCredentials: false,
		AllowOriginFunc: func(origin string) bool {
			return strings.Contains(origin, "hello-storage.vercel.app")
		},
		MaxAge: 12 * time.Hour,
	})
	ProtectedRouter.Use(protectedCorsConfig)
	// Register HTTP route handlers.
	registerRoutes(ProtectedRouter)

	// Register common middleware.
	router.Use(gin.Recovery(), Logger(), middlewares.RateLimitMiddleware(150, 150))
	log.Info("server: common middleware registered")

	router.Any("/api/*action", gin.WrapH(ProtectedRouter))

	config.LoadEnv()

	log.Infof("port: %s", config.Env().AppPort)
	server := &http.Server{
		Addr:           fmt.Sprintf("%s:%s", "0.0.0.0", config.Env().AppPort),
		Handler:        router,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 5 << 30, // 5 GB
	}
	log.Infof("server: listening on %s [%s]", server.Addr, time.Since(start))
	go StartHttp(server)

	// Graceful HTTP server shutdown.
	<-ctx.Done()
	log.Info("server: shutting down")
	err := server.Close()
	if err != nil {
		log.Errorf("server: shutdown failed (%s)", err)
	}
}

// StartHttp starts the web server in http mode.
func StartHttp(s *http.Server) {
	if err := s.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			log.Info("server: shutdown complete")
		} else {
			log.Errorf("server: %s", err)
		}
	}
}
