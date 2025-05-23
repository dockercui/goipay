package db_test

import (
	"context"
	"testing"

	"github.com/chekist32/goipay/internal/db"
	"github.com/chekist32/goipay/test"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
)

func TestCreateUser(t *testing.T) {
	test.RunInTransaction(t, dbConnPool, func(t *testing.T, tx pgx.Tx) {
		ctx := context.Background()
		q := db.New(tx)

		user, err := q.CreateUser(ctx)
		assert.NoError(t, err)
		assert.True(t, user.Valid)
	})
}
