package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/skills"
)

func TestSkillsDiscovery(t *testing.T) {
	// Use the workspace templates as our test workspace
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	workspace := filepath.Join(wd, "workspace")

	loader := skills.NewSkillsLoader(workspace, "", "")
	found := loader.ListSkills()

	if len(found) == 0 {
		t.Fatal("expected skills to be discovered, got 0")
	}

	want := map[string]bool{
		"spreadsheet-helper": false,
		"email-writer":       false,
		"meeting-notes":      false,
	}

	for _, s := range found {
		if _, ok := want[s.Name]; ok {
			want[s.Name] = true
		}
		if s.Source != "workspace" {
			t.Errorf("skill %q: want source=workspace, got %q", s.Name, s.Source)
		}
		if s.Description == "" {
			t.Errorf("skill %q: description is empty", s.Name)
		}
		if s.Path == "" {
			t.Errorf("skill %q: path is empty", s.Name)
		}
	}

	for name, found := range want {
		if !found {
			t.Errorf("skill %q was not discovered", name)
		}
	}
}

func TestSkillsLoadContent(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	workspace := filepath.Join(wd, "workspace")

	loader := skills.NewSkillsLoader(workspace, "", "")

	content, ok := loader.LoadSkill("spreadsheet-helper")
	if !ok {
		t.Fatal("expected spreadsheet-helper skill to load")
	}

	// Content should contain the skill body
	if !strings.Contains(content, "Spreadsheet Helper") {
		t.Error("loaded skill content missing expected heading")
	}
	if !strings.Contains(content, "openpyxl") {
		t.Error("loaded skill content missing expected keyword 'openpyxl'")
	}
}

func TestSkillsSummaryXML(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	workspace := filepath.Join(wd, "workspace")

	loader := skills.NewSkillsLoader(workspace, "", "")
	summary := loader.BuildSkillsSummary()

	if summary == "" {
		t.Fatal("expected non-empty skills summary")
	}

	if !strings.Contains(summary, "<skills>") {
		t.Error("summary missing <skills> tag")
	}
	if !strings.Contains(summary, "<name>spreadsheet-helper</name>") {
		t.Error("summary missing spreadsheet-helper skill")
	}
	if !strings.Contains(summary, "<name>email-writer</name>") {
		t.Error("summary missing email-writer skill")
	}
	if !strings.Contains(summary, "<name>meeting-notes</name>") {
		t.Error("summary missing meeting-notes skill")
	}
	if !strings.Contains(summary, "<source>workspace</source>") {
		t.Error("summary missing workspace source")
	}
}
