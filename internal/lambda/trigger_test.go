package lambda

import (
	"testing"

	"github.com/Kodiqa-Solutions/VaultS3/internal/metadata"
)

func TestExpandTemplate(t *testing.T) {
	cases := []struct {
		tmpl, bucket, key, want string
	}{
		{"{bucket}/out/{key}", "mybucket", "dir/file.jpg", "mybucket/out/dir/file.jpg"},
		{"thumbs/{base}.png", "b", "photo.jpeg", "thumbs/photo.png"},
		{"{base}{ext}", "b", "a/b.txt", "a/b.txt"},
		{"static", "b", "k", "static"},
	}
	for _, c := range cases {
		if got := expandTemplate(c.tmpl, c.bucket, c.key); got != c.want {
			t.Fatalf("expandTemplate(%q,%q,%q)=%q want %q", c.tmpl, c.bucket, c.key, got, c.want)
		}
	}
}

func TestMatchEvent(t *testing.T) {
	cases := []struct {
		patterns []string
		actual   string
		want     bool
	}{
		{[]string{"s3:ObjectCreated:Put"}, "s3:ObjectCreated:Put", true},
		{[]string{"s3:ObjectCreated:*"}, "s3:ObjectCreated:Put", true},
		{[]string{"s3:ObjectCreated:*"}, "s3:ObjectRemoved:Delete", false},
		{[]string{"*"}, "anything", true},
		{[]string{"s3:*"}, "s3:ObjectCreated:Put", true},
		{[]string{"s3:ObjectRemoved:Delete"}, "s3:ObjectCreated:Put", false},
		{nil, "s3:ObjectCreated:Put", false},
	}
	for _, c := range cases {
		if got := matchEvent(c.patterns, c.actual); got != c.want {
			t.Fatalf("matchEvent(%v,%q)=%v want %v", c.patterns, c.actual, got, c.want)
		}
	}
}

func TestMatchFilter(t *testing.T) {
	f := metadata.LambdaTriggerFilter{Prefix: "images/", Suffix: ".jpg"}
	if !matchFilter(f, "images/cat.jpg") {
		t.Fatal("matching prefix+suffix should pass")
	}
	if matchFilter(f, "images/cat.png") {
		t.Fatal("suffix mismatch should fail")
	}
	if matchFilter(f, "videos/cat.jpg") {
		t.Fatal("prefix mismatch should fail")
	}
	if !matchFilter(metadata.LambdaTriggerFilter{}, "anything/goes.bin") {
		t.Fatal("empty filter should match everything")
	}
}
