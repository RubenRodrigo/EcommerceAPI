package auth

import (
	"context"
	"errors"
	"strconv"

	"github.com/rasadov/EcommerceAPI/pkg/contextkeys"
)

var ErrUnauthorized = errors.New("unauthorized")

func GetUserId(ctx context.Context) (string, error) {
	userId, err := GetUserIdInt(ctx)
	if err != nil {
		return "", err
	}
	return strconv.Itoa(userId), nil
}

func GetUserIdInt(ctx context.Context) (int, error) {
	accountId, ok := ctx.Value(contextkeys.UserIDKey).(uint64)
	if !ok {
		return 0, ErrUnauthorized
	}
	return int(accountId), nil
}
