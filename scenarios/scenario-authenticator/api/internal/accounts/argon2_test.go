package accounts

import "testing"

func TestHashVerifyRoundTrip(t *testing.T) {
	hash, err := HashPassword("S3cret-Password")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "S3cret-Password" {
		t.Fatal("password stored in plaintext")
	}
	ok, err := VerifyPassword("S3cret-Password", hash)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Fatal("correct password rejected")
	}
}

func TestHashVerifyWrongPassword(t *testing.T) {
	hash, err := HashPassword("S3cret-Password")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	ok, err := VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if ok {
		t.Fatal("wrong password accepted")
	}
}

func TestHashIsSaltedUnique(t *testing.T) {
	h1, _ := HashPassword("same")
	h2, _ := HashPassword("same")
	if h1 == h2 {
		t.Fatal("identical hashes for same password — salt not applied")
	}
}

func TestVerifyMalformedHash(t *testing.T) {
	if _, err := VerifyPassword("pw", "not-a-phc-string"); err != ErrInvalidHash {
		t.Fatalf("want ErrInvalidHash, got %v", err)
	}
	if _, err := VerifyPassword("pw", "$bcrypt$v=19$..."); err != ErrInvalidHash {
		t.Fatalf("want ErrInvalidHash for non-argon2id, got %v", err)
	}
}
