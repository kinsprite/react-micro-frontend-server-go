package main

import (
	"context"
	"fmt"
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
		ctx.Set(sessionManagerKey, manager)
		store, err := manager.Start(context.Background(), ctx.Writer, ctx.Request)

		if err != nil {
			// reset cookie and restart session (such as in case: err == session.ErrInvalidSessionID)
			ctx.Request.Header.Del("Cookie")
			store, err = manager.Start(context.Background(), ctx.Writer, ctx.Request)
		}

		if err != nil {
			log.Printf("[ERROR] Session start:  %+v\n", err)
		}

		if store != nil {
			ctx.Set(sessionStoreKey, store)
		}

		ctx.Next()
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

func getUserGroups(c *gin.Context) []string {
	store := sessionStoreFromContext(c)

	if store == nil {
		return []string{defaultUserGroup}
	}

	userGroup, ok := store.Get(userGroupKey)

	if ok {
		return userGroup.([]string)
	}

	return []string{defaultUserGroup}
}

func setUserGroups(c *gin.Context, groups []string) error {
	store := sessionStoreFromContext(c)

	if store == nil {
		return fmt.Errorf("No session for the user")
	}

	store.Set(userGroupKey, groups)
	return store.Save()
}
