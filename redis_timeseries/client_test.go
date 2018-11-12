package redis_timeseries

import (
	"github.com/garyburd/redigo/redis"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var client = NewClient("localhost:6379", "test_client")

var defaultDuration, _ = time.ParseDuration("1h")
var defaultMaxSamplesPerChunk uint = 360

func TestCreateKey(t *testing.T) {
	err := client.CreateKey("test_CreateKey", defaultDuration, defaultMaxSamplesPerChunk)
	assert.Equal(t, nil, err)
}

func TestCreateRule(t *testing.T) {
	var destinationKey string
	var err error
	key := "test_CreateRule"
	client.CreateKey(key, defaultDuration, defaultMaxSamplesPerChunk)
	var found bool
	for aggType, aggString := range aggToString {
		destinationKey = "test_CreateRule_dest" + aggString
		client.CreateKey(destinationKey, defaultDuration, defaultMaxSamplesPerChunk)
		err = client.CreateRule(key, aggType, 100, destinationKey)
		assert.Equal(t, nil, err)
		info, _ := client.Info(key)
		found = false
		for _, rule := range info.Rules {
			if aggType == rule.AggType {
				found = true
			}
		}
		assert.True(t, found)
	}
}

func TestClientInfo(t *testing.T) {
	key := "test_INFO"
	destKey := "test_INFO_dest"
	client.CreateKey(key, defaultDuration, defaultMaxSamplesPerChunk)
	client.CreateKey(destKey, defaultDuration, defaultMaxSamplesPerChunk)
	client.CreateRule(key, AvgAggregation, 100, destKey)
	res, err := client.Info(key)
	assert.Equal(t, nil, err)
	expected := KeyInfo{ChunkCount: 1,
		MaxSamplesPerChunk: 360, LastTimestamp: 0, RetentionSecs: 3600,
		Rules: []Rule{{DestKey: destKey, BucketSizeSec: 100, AggType: AvgAggregation}}}
	assert.Equal(t, expected, res)
}

func TestDeleteRule(t *testing.T) {
	key := "test_DELETE"
	destKey := "test_DELETE_dest"
	client.CreateKey(key, defaultDuration, defaultMaxSamplesPerChunk)
	client.CreateKey(destKey, defaultDuration, defaultMaxSamplesPerChunk)
	client.CreateRule(key, AvgAggregation, 100, destKey)
	err := client.DeleteRule(key, destKey)
	assert.Equal(t, nil, err)
	info, _ := client.Info(key)
	assert.Equal(t, 0, len(info.Rules))
	err = client.DeleteRule(key, destKey)
	assert.Equal(t, redis.Error("TSDB: compaction rule does not exist"), err)
}

func TestAdd(t *testing.T) {
	key := "test_ADD"
	now := time.Now().Unix()
	PI := 3.14159265359
	client.CreateKey(key, defaultDuration, defaultMaxSamplesPerChunk)
	err := client.Add(key, now, PI)
	assert.Equal(t, nil, err)
	info, _ := client.Info(key)
	assert.Equal(t, now, info.LastTimestamp)
}

func TestClient_Range(t *testing.T) {
	key := "test_Range"
	client.CreateKey(key, defaultDuration, defaultMaxSamplesPerChunk)
	now := time.Now().Unix()
	pi := 3.14159265359
	halfPi := pi / 2

	client.Add(key, now-2, halfPi)
	client.Add(key, now, pi)

	dataPoints, err := client.Range(key, now-1, now)
	assert.Equal(t, nil, err)
	expected := []DataPoint{{timestamp: now, value: pi}}
	assert.Equal(t, expected, dataPoints)

	dataPoints, err = client.Range(key, now-2, now)
	assert.Equal(t, nil, err)
	expected = []DataPoint{{timestamp: now - 2, value: halfPi}, {timestamp: now, value: pi}}
	assert.Equal(t, expected, dataPoints)

	dataPoints, err = client.Range(key, now-4, now-3)
	assert.Equal(t, nil, err)
	expected = []DataPoint{}
	assert.Equal(t, expected, dataPoints)
}

func TestClient_AggRange(t *testing.T) {
	key := "test_aggRange"
	client.CreateKey(key, defaultDuration, defaultMaxSamplesPerChunk)
	now := time.Now().Unix()
	value := 5.0
	value2 := 6.0

	client.Add(key, now-2, value)
	client.Add(key, now-1, value2)

	dataPoints, err := client.AggRange(key, now-60, now, CountAggregation, 10)
	assert.Equal(t, nil, err)
	assert.Equal(t, 2.0, dataPoints[0].value)
}