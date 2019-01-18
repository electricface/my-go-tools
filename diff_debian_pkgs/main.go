package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
)

func getMap(file string) (map[string]string, error) {
	fh, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(fh)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())

		if len(fields) < 3 {
			continue
		}
		status := fields[0]
		pkgname := fields[1]
		version := fields[2]

		if status == "ii" {
			pkg, arch := splitPkgNameAndArch(pkgname)
			if arch == "i386" {
				continue
			}
			result[pkg] = version
		}
	}

	if scanner.Err() != nil {
		return nil, scanner.Err()
	}

	return result, nil
}

func splitPkgNameAndArch(str string) (string, string) {
	fields := strings.SplitN(str, ":", 2)
	if len(fields) == 1 {
		return fields[0], ""
	}
	return fields[0], fields[1]
}

// get the difference between two maps
func diffVersionInfo(a, b map[string]string) (deleted, added map[string]string, changed map[string][2]string) {
	// from a to b

	deleted = make(map[string]string)
	added = make(map[string]string)
	changed = make(map[string][2]string)

	// keys in a but not in b -> delete
	for ka, va := range a {
		if vb, ok := b[ka]; !ok {
			deleted[ka] = va
		} else {
			if va != vb {
				changed[ka] = [2]string{va, vb}
			}
		}
	}

	// keys in b but not in a -> add
	for kb, vb := range b {
		if _, ok := a[kb]; !ok {
			added[kb] = vb
		}
	}
	return deleted, added, changed
}

type DiffResult struct {
	Deleted, Added map[string]string
	Changed        map[string]VersionChanged
}

type VersionChanged struct {
	From, To string
	Upgrade  bool
}

func (vc *VersionChanged) compare() {
	err := exec.Command("dpkg", "--compare-versions", vc.From, "<", vc.To).Run()
	if err == nil {
		vc.Upgrade = true
	}
}

func diff(fileA, fileB string) (*DiffResult, error) {
	verInfoA, err := getMap(fileA)
	if err != nil {
		return nil, err
	}
	verInfoB, err := getMap(fileB)
	if err != nil {
		return nil, err
	}

	var dr DiffResult
	var changed map[string][2]string
	dr.Deleted, dr.Added, changed = diffVersionInfo(verInfoA, verInfoB)
	verChangedMap := make(map[string]VersionChanged)

	for k, v := range changed {
		verChanged := VersionChanged{
			From: v[0],
			To:   v[1],
		}
		verChanged.compare()

		verChangedMap[k] = verChanged
	}
	dr.Changed = verChangedMap
	return &dr, nil
}

func main() {
	if len(os.Args) == 3 {
		fileA, fileB := os.Args[1], os.Args[2]
		diff, err := diff(fileA, fileB)
		if err != nil {
			log.Fatal(err)
		}

		keys := getSortedKeysMapSS(diff.Deleted)
		for _, k := range keys {
			fmt.Println("D", k, diff.Deleted[k])
		}

		keys = getSortedKeysMapSS(diff.Added)
		for _, k := range keys {
			fmt.Println("A", k, diff.Added[k])
		}

		keys = getSortedKeysMapSVc(diff.Changed)

		for _, k := range keys {
			v := diff.Changed[k]
			upOrDown := "Mv"
			if v.Upgrade {
				upOrDown = "M^"
			}
			fmt.Println(upOrDown, k, v.From, "->", v.To)
		}
	}
}

func getSortedKeysMapSS(m map[string]string) []string {
	var keys = make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func getSortedKeysMapSVc(m map[string]VersionChanged) []string {
	var keys = make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
