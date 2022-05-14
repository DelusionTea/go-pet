package middleware

import (
	"compress/gzip"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-session/session/v3"
	"log"
	"net/http"
	"strings"
)

type gzipWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

func (g *gzipWriter) Write(data []byte) (int, error) {
	return g.writer.Write(data)
}

func GzipEncodeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		if strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") {
			gz, err := gzip.NewWriterLevel(c.Writer, gzip.BestSpeed)
			if err != nil {
				c.AbortWithError(http.StatusBadRequest, err)
				return
			}
			defer gz.Close()
			c.Header("Vary", "Accept-Encoding")
			c.Header("Content-Encoding", "gzip")
			c.Writer = &gzipWriter{c.Writer, gz}
		}
		c.Next()
	}
}
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		store, err := session.Start(context.Background(), c.Writer, c.Request)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, "Server Error")
			log.Println("Server Error 227")
			log.Println(err)
			return
		}
		user, ok := store.Get("user")
		log.Println("user is......", fmt.Sprintf("%v", user))
		if user == nil || (!ok) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		c.Next()
	}
}

func GzipDecodeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.Contains(c.Request.Header.Get("Content-Encoding"), "gzip") {
			return
		}

		r, err := gzip.NewReader(c.Request.Body)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		c.Request.Body = r

		c.Next()

	}
}

//func CookieMiddleware(cfg *conf.Config) gin.HandlerFunc {
//	return func(c *gin.Context) {
//		log.Println("start Cookie Middle ware")
//		defer c.Next()
//		cookie, _ := c.Request.Cookie("userId")
//		log.Println(cookie)
//		encryptor, err := encryption.New(cfg.Key)
//		if err != nil {
//			log.Println("Error")
//			return
//		}
//		if cookie != nil {
//			value, err := encryptor.DecodeUUIDfromString(cookie.Value)
//			if err == nil {
//				log.Println(value)
//				c.Set("userId", value)
//				return
//			}
//		}
//		id, err := uuid.NewV4()
//		if err != nil {
//			return
//		}
//		value := encryptor.EncodeUUIDtoString(id.Bytes())
//		c.SetCookie("userId", value, 864000, "/", cfg.BaseURL, false, false)
//		log.Println("userId")
//		log.Println(id.String())
//		c.Set("userId", id.String())
//	}
//}
