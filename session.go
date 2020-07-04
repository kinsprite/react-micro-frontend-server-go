package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/go-session/session"
)

const sessionManagerKey = "sessionManager"
const sessionStoreKey = "sessionStore"
const sessionCookieName = "sessionId"

// NewSessionMiddleware create a session middleware
func NewSessionMiddleware(opt ...session.Option) gin.HandlerFunc {
	manager := session.NewManager(opt...)

	return func(ctx *gin.Context) {
		// cookie, err := ctx.Cookie(sessionCookieName)
		// fmt.Printf("[Session Cookie IN]: %v\n", cookie)

		ctx.Set(sessionManagerKey, manager)
		store, err := manager.Start(context.Background(), ctx.Writer, ctx.Request)

		if err != nil {
			log.Printf("[ERROR] Session start:  %+v\n", err)
		}

		ctx.Set(sessionStoreKey, store)
		ctx.Next()

		// fmt.Printf("[Session ID]: %+v\n", store.SessionID())
		// fmt.Printf("[Session Cookie OUT]: %v\n", ctx.Writer.Header().Get("Set-Cookie"))
	}
}

func sessionManagerFromContext(ctx *gin.Context) *session.Manager {
	if v, ok := ctx.Get(sessionManagerKey); ok {
		return v.(*session.Manager)
	}

	return nil
}

// sessionStoreFromContext Get session storage from context
func sessionStoreFromContext(ctx *gin.Context) session.Store {
	if v, ok := ctx.Get(sessionStoreKey); ok {
		return v.(session.Store)
	}

	return nil
}

func createSessionMiddleware() gin.HandlerFunc {
	return NewSessionMiddleware(
		session.SetSign([]byte(globalSiteConfig.SessionSign)),
		session.SetCookieName(sessionCookieName),
	)
}

func getUserGroup(c *gin.Context) string {
	store := sessionStoreFromContext(c)

	if store == nil {
		return defaultUserGroup
	}

	userGroup, ok := store.Get(userGroupKey)

	if ok {
		return userGroup.(string)
	}

	return defaultUserGroup
}
