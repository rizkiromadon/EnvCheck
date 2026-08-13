package security

import (
	"strings"
	"testing"

	"github.com/rizkiromadon/envcheck/internal/model"
	"github.com/rizkiromadon/envcheck/internal/parser"
)

func TestScan_DetectsPassword(t *testing.T) {
	pr := parser.Parse("DATABASE_PASSWORD=SuperSecretValue123\n")
	findings := New().Scan(pr.Entries)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(findings), findings)
	}
	if findings[0].RuleID != "generic_password" {
		t.Errorf("expected generic_password rule, got %s", findings[0].RuleID)
	}
}

func TestScan_NeverRevealsFullSecret(t *testing.T) {
	secret := "SuperSecretValue123456"
	pr := parser.Parse("API_KEY=" + secret + "\n")
	findings := New().Scan(pr.Entries)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding")
	}
	if strings.Contains(findings[0].Message, secret) {
		t.Errorf("finding message must not contain the full secret")
	}
	if findings[0].Masked == secret {
		t.Errorf("masked value must not equal the raw secret")
	}
	if !strings.Contains(findings[0].Masked, "*") {
		t.Errorf("masked value should contain masking characters")
	}
}

func TestScan_DetectsAWSAccessKey(t *testing.T) {
	pr := parser.Parse("AWS_KEY=AKIAABCDEFGHIJKLMNOP\n")
	findings := New().Scan(pr.Entries)
	if len(findings) != 1 || findings[0].RuleID != "aws_access_key_id" {
		t.Fatalf("expected aws_access_key_id finding, got %+v", findings)
	}
	if findings[0].Severity != model.SeverityCritical {
		t.Errorf("expected critical severity")
	}
}

func TestScan_DetectsJWT(t *testing.T) {
	jwt := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
	pr := parser.Parse("TOKEN=" + jwt + "\n")
	findings := New().Scan(pr.Entries)
	if len(findings) != 1 || findings[0].RuleID != "jwt_token" {
		t.Fatalf("expected jwt_token finding, got %+v", findings)
	}
}

func TestScan_DetectsPrivateKeyBlock(t *testing.T) {
	pr := parser.Parse("KEY=\"-----BEGIN RSA PRIVATE KEY-----\nMIIExampleKeyData\n-----END RSA PRIVATE KEY-----\"\n")
	findings := New().Scan(pr.Entries)
	if len(findings) != 1 || findings[0].RuleID != "private_key_block" {
		t.Fatalf("expected private_key_block finding, got %+v", findings)
	}
}

func TestScan_DetectsDatabaseURLWithCredentials(t *testing.T) {
	pr := parser.Parse("DATABASE_URL=postgres://user:hunter2@db.example.com:5432/mydb\n")
	findings := New().Scan(pr.Entries)
	if len(findings) != 1 || findings[0].RuleID != "database_url_with_credentials" {
		t.Fatalf("expected database_url_with_credentials finding, got %+v", findings)
	}
}

func TestScan_IgnoresPlaceholders(t *testing.T) {
	pr := parser.Parse("PASSWORD=changeme\nAPI_KEY=your_key_here\nTOKEN=xxxxxxxx\n")
	findings := New().Scan(pr.Entries)
	if len(findings) != 0 {
		t.Errorf("expected no findings for placeholder values, got %+v", findings)
	}
}

func TestScan_IgnoresNonSecretKeys(t *testing.T) {
	pr := parser.Parse("PORT=3000\nNODE_ENV=production\nAPP_NAME=myapp\n")
	findings := New().Scan(pr.Entries)
	if len(findings) != 0 {
		t.Errorf("expected no findings for benign keys, got %+v", findings)
	}
}

func TestMask_NeverEmptyReturnsRawSecret(t *testing.T) {
	cases := []string{"ab", "abcd", "abcdefgh", "abcdefghijklmnopqrstuvwxyz"}
	for _, c := range cases {
		m := Mask(c)
		if m == c {
			t.Errorf("Mask(%q) returned the original value unmasked", c)
		}
	}
}
