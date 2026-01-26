/*
 * api.go - Rute i autentifikacija za EdgeAI dashboard
 *
 * Gin router sa session-based auth za dashboard.
 * Login stranica s formom, session cookie za praćenje prijave.
 */
package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Session storage (in-memory)
var (
	sessions     = make(map[string]sessionData)
	sessionsLock sync.RWMutex
)

type sessionData struct {
	username  string
	createdAt time.Time
}

// Generira random session token
func generateSessionToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Session Auth middleware - provjerava session cookie
func SessionAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("session")
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		sessionsLock.RLock()
		session, exists := sessions[cookie]
		sessionsLock.RUnlock()

		if !exists || time.Since(session.createdAt) > 24*time.Hour {
			// Session expired ili ne postoji
			sessionsLock.Lock()
			delete(sessions, cookie)
			sessionsLock.Unlock()
			c.SetCookie("session", "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		c.Set("username", session.username)
		c.Next()
	}
}

// SetupRouter - postavlja sve rute za backend
func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")   // HTML template za dashboard
	r.Static("/static", "./static") // CSS/JS fajlovi

	// Login stranica (GET)
	r.GET("/login", func(c *gin.Context) {
		// Ako je već prijavljen, preusmjeri na dashboard
		if cookie, err := c.Cookie("session"); err == nil {
			sessionsLock.RLock()
			_, exists := sessions[cookie]
			sessionsLock.RUnlock()
			if exists {
				c.Redirect(http.StatusFound, "/")
				return
			}
		}
		c.HTML(http.StatusOK, "login.html", gin.H{"Error": ""})
	})

	// Login handler (POST)
	r.POST("/login", func(c *gin.Context) {
		username := c.PostForm("username")
		password := c.PostForm("password")

		// Provjera korisnika i lozinke
		if username != cfg.DashboardUser {
			c.HTML(http.StatusOK, "login.html", gin.H{"Error": "Pogrešno korisničko ime ili lozinka"})
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(cfg.DashboardPassHash), []byte(password)) != nil {
			c.HTML(http.StatusOK, "login.html", gin.H{"Error": "Pogrešno korisničko ime ili lozinka"})
			return
		}

		// Kreiranje sessiona
		token := generateSessionToken()
		sessionsLock.Lock()
		sessions[token] = sessionData{
			username:  username,
			createdAt: time.Now(),
		}
		sessionsLock.Unlock()

		// Postavi cookie (24 sata, httpOnly)
		c.SetCookie("session", token, 86400, "/", "", false, true)
		c.Redirect(http.StatusFound, "/")
	})

	// Logout handler
	r.GET("/logout", func(c *gin.Context) {
		if cookie, err := c.Cookie("session"); err == nil {
			sessionsLock.Lock()
			delete(sessions, cookie)
			sessionsLock.Unlock()
		}
		c.SetCookie("session", "", -1, "/", "", false, true)
		c.Redirect(http.StatusFound, "/login")
	})

	// Zaštićene rute (dashboard)
	authorized := r.Group("/", SessionAuthMiddleware())
	authorized.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// Javne rute
	r.GET("/api/faults", func(c *gin.Context) {
		c.JSON(http.StatusOK, GetFaultEvents())
	})
	r.GET("/ws", WebSocketHandler)
	return r
}
