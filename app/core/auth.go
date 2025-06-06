package core

import (
	"context"
	"time"
)

type ISessionStorage interface {
	NewSession(user Authenticable) (string, time.Time, error)
	IsLoggedIn(ctx context.Context) bool
	Logout(ctx context.Context)
	CurrentUser(ctx context.Context) (Authenticable, error)
	ClearExpiredSessions()
}

type Authenticable interface {
	GetUsername() string
	GetPassword() string
	GetRole() UserRole
}

type UserRole string

type IUserService interface {
	Get(id uint64) (Authenticable, error)
	GetByUsername(username string) (Authenticable, error)
	Save(authEntity Authenticable) error
}

type AuthService interface {
	CurrentUser(ctx context.Context) (Authenticable, error)
}

type Policy[T any] interface {
	AddFilter(ctx context.Context, builder IOrm[T]) (bool, IOrm[T])
	AddFilterForBuilder(ctx context.Context, builder IQueryBuilder) (bool, IQueryBuilder)
	Create(ctx context.Context) bool
	ShowAny(ctx context.Context) bool
	Show(ctx context.Context, model any) bool
	Update(ctx context.Context, model any) bool
	Delete(ctx context.Context, model any) bool
}
