package storage

import (
	"IO_BOUND/task"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type Storage struct {
	db *redis.Client
}

func newRedisClient(ctx context.Context, address, user, password string, db int) (*redis.Client, error) {
	const op = "storage.newRedisClient"

	opts := &redis.Options{
		Addr:            address,
		ClientName:      user,
		Password:        password,
		DB:              db,
		PoolSize:        100,
		MinIdleConns:    10,
		ConnMaxIdleTime: 5 * time.Minute,
	}

	client := redis.NewClient(opts)

	if err := client.ConfigSet(ctx, "maxmemory", "256mb").Err(); err != nil {
		return nil, fmt.Errorf("%s: unable to set memory size: %w", op, err)
	}
	if err := client.ConfigSet(ctx, "maxmemory-policy", "allkeys-lru").Err(); err != nil {
		return nil, fmt.Errorf("%s: unable set memory policy: %w", op, err)
	}

	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("%s: connection failed: %w", op, err)
	}

	return client, nil
}

func New(ctx context.Context, address, user, password string) (*Storage, error) {
	const op = "storage.NewStorage"
	client, err := newRedisClient(ctx, address, user, password, 0)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &Storage{db: client}, nil
}

func (strg *Storage) Get(ctx context.Context, key string) ([]byte, error) {
	return strg.db.Get(ctx, key).Bytes()
}
func (strg *Storage) Save(ctx context.Context, key string, t *task.Task) error {
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return strg.db.Set(ctx, key, data, 0).Err()
}
func (strg *Storage) Delete(ctx context.Context, key string) error {
	deleted, err := strg.db.Del(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("error while deleting key: %v", err)
	}
	if deleted == 1 {
		log.Println("key successfully deleted")
	} else {
		return fmt.Errorf("key not found")
	}
	return nil
}
func (strg *Storage) Close() {
	strg.db.Conn().Close()
}

func (strg *Storage) GetAll(ctx context.Context) {
	keys, err := strg.db.Keys(ctx, "*").Result()
	if err != nil {
		log.Fatalf("error geting keys: %v", err)
	}

	fmt.Printf("Found %d keys:\n", len(keys))
	for _, key := range keys {
		keyType, err := strg.db.Type(ctx, key).Result()
		if err != nil {
			log.Printf("error geting key type %s: %v", key, err)
			continue
		}
		fmt.Printf("Key: %s, Type: %s\n", key, keyType)
		val, err := strg.db.Get(ctx, key).Result()
		if err != nil {
			log.Printf("error geting key value %s: %v", key, err)
			continue
		}
		fmt.Printf("Value: %v\n", val)
	}
}
