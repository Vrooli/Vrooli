#!/usr/bin/env bats
# Redis Mock Test Suite
#
# Comprehensive tests for the Redis mock implementation
# Tests all Redis commands, state management, error injection,
# and BATS compatibility features

# Source trash module for safe test cleanup
MOCK_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${MOCK_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Test environment setup
setup() {
    # Set up test directory
    export TEST_DIR="$BATS_TEST_TMPDIR/redis-mock-test"
    mkdir -p "$TEST_DIR"
    cd "$TEST_DIR"
    
    # Set up mock directory
    export MOCK_DIR="${BATS_TEST_DIRNAME}"
    
    # Configure Redis mock state directory
    export REDIS_MOCK_STATE_DIR="$TEST_DIR/redis-state"
    mkdir -p "$REDIS_MOCK_STATE_DIR"
    
    # Source the Redis mock
    source "$MOCK_DIR/redis.sh"
    
    # Reset Redis mock to clean state
    mock::redis::reset
}

teardown() {
    # Clean up test directory
    trash::safe_remove "$TEST_DIR" --test-cleanup
}

# Helper functions for assertions
assert_success() {
    if [[ "$status" -ne 0 ]]; then
        echo "Expected success but got status $status" >&2
        echo "Output: $output" >&2
        return 1
    fi
}

assert_failure() {
    if [[ "$status" -eq 0 ]]; then
        echo "Expected failure but got success" >&2
        echo "Output: $output" >&2
        return 1
    fi
}

assert_output() {
    local expected="$1"
    if [[ "$1" == "--partial" ]]; then
        expected="$2"
        if [[ ! "$output" =~ "$expected" ]]; then
            echo "Expected output to contain: $expected" >&2
            echo "Actual output: $output" >&2
            return 1
        fi
    elif [[ "$1" == "--regexp" ]]; then
        expected="$2"
        if [[ ! "$output" =~ $expected ]]; then
            echo "Expected output to match: $expected" >&2
            echo "Actual output: $output" >&2
            return 1
        fi
    else
        if [[ "$output" != "$expected" ]]; then
            echo "Expected: $expected" >&2
            echo "Actual: $output" >&2
            return 1
        fi
    fi
}

refute_output() {
    local pattern="$2"
    if [[ "$1" == "--partial" ]] || [[ "$1" == "--regexp" ]]; then
        if [[ "$output" =~ "$pattern" ]]; then
            echo "Expected output NOT to contain: $pattern" >&2
            echo "Actual output: $output" >&2
            return 1
        fi
    fi
}

assert_line() {
    local index expected
    if [[ "$1" == "--index" ]]; then
        index="$2"
        expected="$3"
        local lines=()
        while IFS= read -r line; do
            lines+=("$line")
        done <<< "$output"
        
        if [[ "${lines[$index]}" != "$expected" ]]; then
            echo "Line $index mismatch" >&2
            echo "Expected: $expected" >&2
            echo "Actual: ${lines[$index]}" >&2
            return 1
        fi
    fi
}

# Basic connectivity tests
@test "redis-cli: basic ping command" {
    run redis-cli ping
    assert_success
    assert_output "PONG"
}

@test "redis-cli: ping with custom message" {
    run redis-cli ping "Hello World"
    assert_success
    assert_output "Hello World"
}

@test "redis-cli: connection refused when disconnected" {
    mock::redis::set_connected "false"
    
    run redis-cli ping
    assert_failure
    assert_output --partial "Connection refused"
}

@test "redis-cli: error injection - connection timeout" {
    mock::redis::set_error "connection_timeout"
    
    run redis-cli ping
    assert_failure
    assert_output --partial "Connection timed out"
}

@test "redis-cli: error injection - auth failed" {
    mock::redis::set_error "auth_failed"
    
    run redis-cli ping
    assert_failure
    assert_output "NOAUTH Authentication required."
}

@test "redis-cli: error injection - loading" {
    mock::redis::set_error "loading"
    
    run redis-cli ping
    assert_failure
    assert_output "LOADING Redis is loading the dataset in memory"
}

# String operations tests
@test "redis-cli: SET and GET basic operations" {
    run redis-cli set mykey "hello world"
    assert_success
    assert_output "OK"
    
    run redis-cli get mykey
    assert_success
    assert_output "hello world"
}

@test "redis-cli: GET non-existent key returns nil" {
    run redis-cli get nonexistent
    assert_success
    assert_output "(nil)"
}

@test "redis-cli: SET with EX option (expiration in seconds)" {
    run redis-cli set tempkey "temp value" ex 60
    assert_success
    assert_output "OK"
    
    run redis-cli ttl tempkey
    assert_success
    # TTL should be between 59 and 60 seconds
    [[ "${output#(integer) }" -ge 59 ]] && [[ "${output#(integer) }" -le 60 ]]
}

@test "redis-cli: SET with PX option (expiration in milliseconds)" {
    run redis-cli set tempkey "temp value" px 60000
    assert_success
    assert_output "OK"
    
    run redis-cli ttl tempkey
    assert_success
    # TTL should be around 60 seconds
    [[ "${output#(integer) }" -ge 59 ]] && [[ "${output#(integer) }" -le 60 ]]
}

@test "redis-cli: SET with NX option (only if not exists)" {
    run redis-cli set mykey "first value"
    assert_success
    assert_output "OK"
    
    run redis-cli set mykey "second value" nx
    assert_success
    assert_output "(nil)"
    
    run redis-cli get mykey
    assert_success
    assert_output "first value"
}

@test "redis-cli: SET with XX option (only if exists)" {
    run redis-cli set mykey "new value" xx
    assert_success
    assert_output "(nil)"
    
    run redis-cli set mykey "first value"
    assert_success
    assert_output "OK"
    
    run redis-cli set mykey "updated value" xx
    assert_success
    assert_output "OK"
    
    run redis-cli get mykey
    assert_success
    assert_output "updated value"
}

@test "redis-cli: DEL single key" {
    redis-cli set key1 "value1"
    
    run redis-cli del key1
    assert_success
    assert_output "(integer) 1"
    
    run redis-cli get key1
    assert_success
    assert_output "(nil)"
}

@test "redis-cli: DEL multiple keys" {
    redis-cli set key1 "value1"
    redis-cli set key2 "value2"
    redis-cli set key3 "value3"
    
    run redis-cli del key1 key2 key4
    assert_success
    assert_output "(integer) 2"
}

@test "redis-cli: EXISTS command" {
    redis-cli set key1 "value1"
    redis-cli set key2 "value2"
    
    run redis-cli exists key1
    assert_success
    assert_output "(integer) 1"
    
    run redis-cli exists key1 key2 key3
    assert_success
    assert_output "(integer) 2"
}

@test "redis-cli: EXPIRE and TTL commands" {
    redis-cli set mykey "value"
    
    run redis-cli expire mykey 10
    assert_success
    assert_output "(integer) 1"
    
    run redis-cli ttl mykey
    assert_success
    [[ "${output#(integer) }" -ge 9 ]] && [[ "${output#(integer) }" -le 10 ]]
    
    run redis-cli expire nonexistent 10
    assert_success
    assert_output "(integer) 0"
}

@test "redis-cli: TTL returns -1 for keys without expiration" {
    redis-cli set mykey "value"
    
    run redis-cli ttl mykey
    assert_success
    assert_output "(integer) -1"
}

@test "redis-cli: TTL returns -2 for non-existent keys" {
    run redis-cli ttl nonexistent
    assert_success
    assert_output "(integer) -2"
}

@test "redis-cli: KEYS pattern matching" {
    redis-cli set user:1000 "john"
    redis-cli set user:1001 "mary"
    redis-cli set product:100 "laptop"
    
    run redis-cli keys "user:*"
    assert_success
    assert_output --regexp "user:1000"
    assert_output --regexp "user:1001"
    refute_output --regexp "product:100"
    
    run redis-cli keys "*"
    assert_success
    assert_output --regexp "user:1000"
    assert_output --regexp "user:1001"
    assert_output --regexp "product:100"
}

@test "redis-cli: FLUSHDB clears all data" {
    redis-cli set key1 "value1"
    redis-cli lpush list1 "item1"
    redis-cli sadd set1 "member1"
    
    run redis-cli flushdb
    assert_success
    assert_output "OK"
    
    run redis-cli keys "*"
    assert_success
    assert_output "(empty array)"
}

# List operations tests
@test "redis-cli: LPUSH and LLEN" {
    run redis-cli lpush mylist "world"
    assert_success
    assert_output "(integer) 1"
    
    run redis-cli lpush mylist "hello"
    assert_success
    assert_output "(integer) 2"
    
    run redis-cli llen mylist
    assert_success
    assert_output "(integer) 2"
}

@test "redis-cli: RPUSH adds to end of list" {
    redis-cli lpush mylist "first"
    
    run redis-cli rpush mylist "last"
    assert_success
    assert_output "(integer) 2"
    
    run redis-cli lrange mylist 0 -1
    assert_success
    assert_line --index 0 '1) "first"'
    assert_line --index 1 '2) "last"'
}

@test "redis-cli: LPOP removes from beginning" {
    redis-cli lpush mylist "second"
    redis-cli lpush mylist "first"
    
    run redis-cli lpop mylist
    assert_success
    assert_output "first"
    
    run redis-cli llen mylist
    assert_success
    assert_output "(integer) 1"
}

@test "redis-cli: RPOP removes from end" {
    redis-cli rpush mylist "first"
    redis-cli rpush mylist "second"
    
    run redis-cli rpop mylist
    assert_success
    assert_output "second"
    
    run redis-cli llen mylist
    assert_success
    assert_output "(integer) 1"
}

@test "redis-cli: LRANGE with positive indices" {
    redis-cli rpush mylist "one"
    redis-cli rpush mylist "two"
    redis-cli rpush mylist "three"
    
    run redis-cli lrange mylist 0 1
    assert_success
    assert_line --index 0 '1) "one"'
    assert_line --index 1 '2) "two"'
}

@test "redis-cli: LRANGE with negative indices" {
    redis-cli rpush mylist "one"
    redis-cli rpush mylist "two"
    redis-cli rpush mylist "three"
    
    run redis-cli lrange mylist -2 -1
    assert_success
    assert_line --index 0 '1) "two"'
    assert_line --index 1 '2) "three"'
}

@test "redis-cli: LRANGE on empty list" {
    run redis-cli lrange mylist 0 -1
    assert_success
    assert_output "(empty array)"
}

# Set operations tests
@test "redis-cli: SADD adds unique members" {
    run redis-cli sadd myset "hello"
    assert_success
    assert_output "(integer) 1"
    
    run redis-cli sadd myset "world"
    assert_success
    assert_output "(integer) 1"
    
    run redis-cli sadd myset "hello"
    assert_success
    assert_output "(integer) 0"
}

@test "redis-cli: SADD multiple members at once" {
    run redis-cli sadd myset "a" "b" "c" "b"
    assert_success
    assert_output "(integer) 3"
}

@test "redis-cli: SREM removes members" {
    redis-cli sadd myset "one" "two" "three"
    
    run redis-cli srem myset "two" "four"
    assert_success
    assert_output "(integer) 1"
    
    run redis-cli smembers myset
    assert_success
    refute_output --regexp "two"
}

@test "redis-cli: SMEMBERS lists all members" {
    redis-cli sadd myset "apple" "banana" "cherry"
    
    run redis-cli smembers myset
    assert_success
    assert_output --regexp "apple"
    assert_output --regexp "banana"
    assert_output --regexp "cherry"
}

@test "redis-cli: SISMEMBER checks membership" {
    redis-cli sadd myset "hello"
    
    run redis-cli sismember myset "hello"
    assert_success
    assert_output "(integer) 1"
    
    run redis-cli sismember myset "world"
    assert_success
    assert_output "(integer) 0"
}

# Hash operations tests
@test "redis-cli: HSET and HGET" {
    run redis-cli hset myhash field1 "value1"
    assert_success
    assert_output "(integer) 1"
    
    run redis-cli hget myhash field1
    assert_success
    assert_output "value1"
    
    run redis-cli hget myhash field2
    assert_success
    assert_output "(nil)"
}

@test "redis-cli: HDEL removes fields" {
    redis-cli hset myhash field1 "value1"
    redis-cli hset myhash field2 "value2"
    
    run redis-cli hdel myhash field1 field3
    assert_success
    assert_output "(integer) 1"
    
    run redis-cli hget myhash field1
    assert_success
    assert_output "(nil)"
}

@test "redis-cli: HGETALL returns all fields and values" {
    redis-cli hset myhash name "John"
    redis-cli hset myhash age "30"
    
    run redis-cli hgetall myhash
    assert_success
    assert_output --regexp "name"
    assert_output --regexp "John"
    assert_output --regexp "age"
    assert_output --regexp "30"
}

@test "redis-cli: HGETALL on empty hash" {
    run redis-cli hgetall myhash
    assert_success
    assert_output "(empty array)"
}

# Pub/Sub operations tests
@test "redis-cli: PUBLISH returns subscriber count" {
    run redis-cli publish mychannel "Hello subscribers"
    assert_success
    assert_output "(integer) 0"
}

@test "redis-cli: SUBSCRIBE shows subscription confirmation" {
    run redis-cli subscribe mychannel
    assert_success
    assert_output --partial "Reading messages... (press Ctrl-C to quit)"
    assert_output --partial '"subscribe"'
    assert_output --partial "mychannel"
}

# Transaction operations tests
@test "redis-cli: MULTI/EXEC basic transaction" {
    run redis-cli multi
    assert_success
    assert_output "OK"
    
    run redis-cli set key1 "value1"
    assert_success
    assert_output "QUEUED"
    
    run redis-cli set key2 "value2"
    assert_success
    assert_output "QUEUED"
    
    run redis-cli exec
    assert_success
    assert_line --index 0 '1) OK'
    assert_line --index 1 '2) OK'
    
    # Verify the commands were executed
    run redis-cli get key1
    assert_success
    assert_output "value1"
}

@test "redis-cli: DISCARD cancels transaction" {
    redis-cli multi
    redis-cli set key1 "value1"
    
    run redis-cli discard
    assert_success
    assert_output "OK"
    
    run redis-cli get key1
    assert_success
    assert_output "(nil)"
}

@test "redis-cli: EXEC without MULTI fails" {
    run redis-cli exec
    assert_failure
    assert_output --partial "EXEC without MULTI"
}

@test "redis-cli: DISCARD without MULTI fails" {
    run redis-cli discard
    assert_failure
    assert_output --partial "DISCARD without MULTI"
}

# CONFIG operations tests
@test "redis-cli: CONFIG GET" {
    run redis-cli config get maxmemory
    assert_success
    assert_line --index 0 '1) "maxmemory"'
    assert_line --index 1 '2) "0"'
    
    run redis-cli config get unknown
    assert_success
    assert_output "(empty array)"
}

@test "redis-cli: CONFIG SET" {
    run redis-cli config set maxmemory 1000000
    assert_success
    assert_output "OK"
}

# INFO command tests
@test "redis-cli: INFO without section" {
    run redis-cli info
    assert_success
    assert_output --partial "# Server"
    assert_output --partial "redis_version:"
    assert_output --partial "# Clients"
    assert_output --partial "# Memory"
    assert_output --partial "# Replication"
}

@test "redis-cli: INFO with specific section" {
    run redis-cli info server
    assert_success
    assert_output --partial "# Server"
    assert_output --partial "redis_version:"
    refute_output --partial "# Clients"
}

# Command line argument parsing tests
@test "redis-cli: custom host and port" {
    run redis-cli -h redis.example.com -p 6380 ping
    assert_success
    assert_output "PONG"
}

@test "redis-cli: --raw option" {
    run redis-cli --raw get somekey
    assert_success
    assert_output "(nil)"
}

@test "redis-cli: interactive mode not supported" {
    run redis-cli
    assert_failure
    assert_output --partial "Interactive mode not supported"
}

# State persistence tests
@test "redis state persistence across subshells" {
    # Set data in parent shell
    redis-cli set persistkey "persistvalue"
    mock::redis::save_state
    
    # Verify in subshell
    output=$(
        source "$MOCK_DIR/redis.sh"
        redis-cli get persistkey
    )
    [[ "$output" == "persistvalue" ]]
}

@test "redis state file creation and loading" {
    redis-cli set statekey "statevalue"
    redis-cli lpush statelist "item1"
    mock::redis::save_state
    
    # Check state file exists
    [[ -f "$REDIS_MOCK_STATE_DIR/redis-state.sh" ]]
    
    # Reset without saving state, then reload
    mock::redis::reset false
    mock::redis::load_state
    
    run redis-cli get statekey
    assert_success
    assert_output "statevalue"
    
    run redis-cli llen statelist
    assert_success
    assert_output "(integer) 1"
}

# Test helper functions
@test "mock::redis::reset clears all data" {
    redis-cli set key1 "value1"
    redis-cli lpush list1 "item1"
    redis-cli sadd set1 "member1"
    mock::redis::set_error "connection_timeout"
    
    mock::redis::reset
    
    run redis-cli keys "*"
    assert_success
    assert_output "(empty array)"
    
    # Error mode should be cleared
    run redis-cli ping
    assert_success
    assert_output "PONG"
}

@test "mock::redis::assert_key_exists" {
    redis-cli set mykey "value"
    
    run mock::redis::assert_key_exists "mykey"
    assert_success
    
    run mock::redis::assert_key_exists "nonexistent"
    assert_failure
    assert_output --partial "Key 'nonexistent' does not exist"
}

@test "mock::redis::assert_key_value" {
    redis-cli set mykey "hello"
    
    run mock::redis::assert_key_value "mykey" "hello"
    assert_success
    
    run mock::redis::assert_key_value "mykey" "world"
    assert_failure
    assert_output --partial "Key 'mykey' value mismatch"
    assert_output --partial "Expected: 'world'"
    assert_output --partial "Actual: 'hello'"
}

@test "mock::redis::assert_list_length" {
    redis-cli rpush mylist "a" "b" "c"
    
    run mock::redis::assert_list_length "mylist" 3
    assert_success
    
    run mock::redis::assert_list_length "mylist" 2
    assert_failure
    assert_output --partial "List 'mylist' length mismatch"
}

@test "mock::redis::assert_set_member" {
    redis-cli sadd myset "apple" "banana"
    
    run mock::redis::assert_set_member "myset" "apple"
    assert_success
    
    run mock::redis::assert_set_member "myset" "cherry"
    assert_failure
    assert_output --partial "Member 'cherry' not in set 'myset'"
}

@test "mock::redis::dump_state shows current state" {
    redis-cli set key1 "value1"
    redis-cli lpush list1 "item1"
    mock::redis::set_config "port" "6380"
    
    run mock::redis::dump_state
    assert_success
    assert_output --partial "Redis Mock State"
    assert_output --partial "key1: value1"
    assert_output --partial "list1:"
    assert_output --partial "port: 6380"
}

# Complex scenario tests
@test "redis: expired keys are automatically removed" {
    # Set a key with 1 second expiration
    redis-cli set tempkey "tempvalue" ex 1
    
    # Key should exist immediately
    run redis-cli get tempkey
    assert_success
    assert_output "tempvalue"
    
    # Wait for expiration
    sleep 2
    
    # Key should be gone
    run redis-cli get tempkey
    assert_success
    assert_output "(nil)"
    
    # TTL should return -2
    run redis-cli ttl tempkey
    assert_success
    assert_output "(integer) -2"
}

@test "redis: transaction with multiple data types" {
    redis-cli multi
    redis-cli set string:key "value"
    redis-cli lpush list:key "item1"
    redis-cli sadd set:key "member1"
    redis-cli hset hash:key field1 "value1"
    
    run redis-cli exec
    assert_success
    assert_line --index 0 '1) OK'
    assert_line --index 1 '2) (integer) 1'
    assert_line --index 2 '3) (integer) 1'
    assert_line --index 3 '4) (integer) 1'
    
    # Verify all operations succeeded
    run redis-cli get string:key
    assert_output "value"
    run redis-cli llen list:key
    assert_output "(integer) 1"
    run redis-cli sismember set:key "member1"
    assert_output "(integer) 1"
    run redis-cli hget hash:key field1
    assert_output "value1"
}

@test "redis: DEL works across all data types" {
    redis-cli set string:key "value"
    redis-cli lpush list:key "item"
    redis-cli sadd set:key "member"
    redis-cli hset hash:key field "value"
    
    run redis-cli del string:key list:key set:key hash:key nonexistent
    assert_success
    assert_output "(integer) 4"
    
    # Verify all are deleted
    run redis-cli exists string:key list:key set:key hash:key
    assert_success
    assert_output "(integer) 0"
}

# Error handling tests
@test "redis: commands with wrong number of arguments" {
    run redis-cli set
    assert_failure
    assert_output --partial "wrong number of arguments"
    
    run redis-cli get
    assert_failure
    assert_output --partial "wrong number of arguments"
    
    run redis-cli lpush mylist
    assert_failure
    assert_output --partial "wrong number of arguments"
}

@test "redis: unknown command" {
    run redis-cli unknowncommand arg1 arg2
    assert_failure
    assert_output --partial "unknown command 'UNKNOWNCOMMAND'"
}

# Edge cases
@test "redis: empty string values" {
    run redis-cli set emptykey ""
    assert_success
    assert_output "OK"
    
    run redis-cli get emptykey
    assert_success
    assert_output ""
}

@test "redis: keys with special characters" {
    run redis-cli set "key:with:colons" "value"
    assert_success
    
    run redis-cli set "key with spaces" "value"
    assert_success
    
    run redis-cli get "key:with:colons"
    assert_success
    assert_output "value"
    
    run redis-cli get "key with spaces"
    assert_success
    assert_output "value"
}

@test "redis: list operations on empty lists" {
    run redis-cli lpop emptylist
    assert_success
    assert_output "(nil)"
    
    run redis-cli rpop emptylist
    assert_success
    assert_output "(nil)"
    
    run redis-cli llen emptylist
    assert_success
    assert_output "(integer) 0"
}