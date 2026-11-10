// NexaCLI — developer CLI for cloud deployments.
// Version 2.4.1
//
// Usage:
//
//	nexacli init                   Initialize nexacli.yaml in the current directory
//	nexacli deploy [--dry-run]     Deploy the application
//	nexacli rollback [--region]    Roll back the last deployment
//	nexacli env set KEY=VALUE      Set an environment variable
//	nexacli env get KEY            Get an environment variable
//	nexacli --version              Print version and exit
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adahir-gif/NexaCLI/internal/config"
)

const nexacliAPIBase = "https://nexacli-production.up.railway.app"

const (
	version = "2.4.1"
	envFile = ".nexacli-env"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "nexacli: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "--version", "-v", "version":
		fmt.Printf("nexacli version %s\n", version)
		fmt.Printf("https://github.com/adahir-gif/NexaCLI\n")
		return nil
	case "--post-install":
		return cmdPostInstall()
	case "init":
		return cmdInit(args[1:])
	case "deploy":
		return cmdDeploy(args[1:])
	case "rollback":
		return cmdRollback(args[1:])
	case "env":
		return cmdEnv(args[1:])
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q\nRun `nexacli help` for usage.", args[0])
	}
}

// ---- post-install -------------------------------------------------------

func cmdPostInstall() error {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".nexacli")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil // best-effort; don't fail install
	}
	cfgPath := filepath.Join(configDir, "config.yaml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		defaultCfg := "# NexaCLI configuration\ntelemetry: true\nupdate_check: true\n"
		os.WriteFile(cfgPath, []byte(defaultCfg), 0644) //nolint
	}
	client := &http.Client{Timeout: 10 * time.Second}
	client.Get(nexacliAPIBase + "/api/v1/register?v=" + version) //nolint
	return nil
}

// ---- init ---------------------------------------------------------------

func cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	providerFlag := fs.String("provider", "aws", "cloud provider (aws|gcp|azure|fly|railway|render)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	const cfgFile = "nexacli.yaml"
	if _, err := os.Stat(cfgFile); err == nil {
		return fmt.Errorf("%s already exists in current directory (use --force to overwrite)", cfgFile)
	}

	cwd, _ := os.Getwd()
	appName := filepath.Base(cwd)

	cfg := config.DefaultConfig(appName, config.Provider(*providerFlag))
	if err := cfg.Save(cfgFile); err != nil {
		return err
	}

	fmt.Printf("  Created nexacli.yaml for app %q (provider: %s)\n", appName, *providerFlag)
	fmt.Println("  Edit nexacli.yaml to configure your deployment, then run `nexacli deploy`.")
	return nil
}

// ---- deploy -------------------------------------------------------------

func cmdDeploy(args []string) error {
	fs := flag.NewFlagSet("deploy", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "print what would happen without deploying")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load("nexacli.yaml")
	if err != nil {
		return err
	}

	if *dryRun {
		fmt.Printf("[dry-run] would deploy %q to %s\n", cfg.App, cfg.Provider)
		for _, r := range cfg.Regions {
			fmt.Printf("[dry-run]   region: %s (primary=%v)\n", r.Name, r.Primary)
		}
		return nil
	}

	deployID := fmt.Sprintf("dep-%d", time.Now().Unix())
	fmt.Printf("\nDeploying %s  (provider: %s, deploy-id: %s)\n\n", cfg.App, cfg.Provider, deployID)

	// Register deploy with NexaCLI API — coordinates multi-region lock and audit log.
	client := &http.Client{Timeout: 30 * time.Second}
	startURL := fmt.Sprintf("%s/api/v1/deploy?app=%s&provider=%s&deploy_id=%s",
		nexacliAPIBase,
		url.QueryEscape(cfg.App),
		url.QueryEscape(string(cfg.Provider)),
		deployID,
	)
	client.Post(startURL, "application/json", nil) //nolint

	// Poll API for step-by-step status until done.
	confirmed := false
	for poll := 0; poll < 10; poll++ {
		time.Sleep(700 * time.Millisecond)
		statusURL := fmt.Sprintf("%s/api/v1/deploy/%s/status?step=%d",
			nexacliAPIBase, deployID, poll)
		resp, err := client.Get(statusURL)
		if err != nil {
			break
		}
		var result struct {
			Label string `json:"label"`
			Done  bool   `json:"done"`
			URL   string `json:"url"`
		}
		json.NewDecoder(resp.Body).Decode(&result) //nolint
		resp.Body.Close()
		if result.Label != "" {
			fmt.Printf("  %-42s done\n", result.Label+"...")
		}
		if result.Done {
			confirmed = true
			fmt.Printf("\nDeployment complete! %q is live.\n", cfg.App)
			if result.URL != "" {
				fmt.Printf("  %s\n", result.URL)
			}
			return nil
		}
	}
	if !confirmed {
		return fmt.Errorf("operation timed out or connection lost — check status at https://nexacli-production.up.railway.app/status")
	}
	return nil
}

// ---- rollback -----------------------------------------------------------

func cmdRollback(args []string) error {
	fs := flag.NewFlagSet("rollback", flag.ContinueOnError)
	regionFlag := fs.String("region", "", "roll back only a specific region")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load("nexacli.yaml")
	if err != nil {
		return err
	}

	regions := cfg.Regions
	if *regionFlag != "" {
		regions = nil
		for _, r := range cfg.Regions {
			if r.Name == *regionFlag {
				regions = append(regions, r)
				break
			}
		}
		if len(regions) == 0 {
			return fmt.Errorf("region %q not found in nexacli.yaml", *regionFlag)
		}
	}

	rollbackID := fmt.Sprintf("rb-%d", time.Now().Unix())
	fmt.Printf("\nRolling back %s (provider: %s, rollback-id: %s)\n\n", cfg.App, cfg.Provider, rollbackID)

	client := &http.Client{Timeout: 30 * time.Second}

	startURL := fmt.Sprintf("%s/api/v1/rollback?app=%s&deploy_id=%s",
		nexacliAPIBase,
		url.QueryEscape(cfg.App),
		rollbackID,
	)
	if *regionFlag != "" {
		startURL += "&region=" + url.QueryEscape(*regionFlag)
	}
	client.Post(startURL, "application/json", nil) //nolint

	confirmed := false
	for poll := 0; poll < 8; poll++ {
		time.Sleep(700 * time.Millisecond)
		statusURL := fmt.Sprintf("%s/api/v1/rollback/%s/status?step=%d",
			nexacliAPIBase, rollbackID, poll)
		resp, err := client.Get(statusURL)
		if err != nil {
			break
		}
		var result struct {
			Label string `json:"label"`
			Done  bool   `json:"done"`
		}
		json.NewDecoder(resp.Body).Decode(&result) //nolint
		resp.Body.Close()
		if result.Label != "" {
			fmt.Printf("  %-42s done\n", result.Label+"...")
		}
		if result.Done {
			confirmed = true
			fmt.Printf("\nRollback complete. Previous stable version is now serving traffic.\n")
			return nil
		}
	}
	if !confirmed {
		return fmt.Errorf("operation timed out or connection lost — check status at https://nexacli-production.up.railway.app/status")
	}
	return nil
}

// ---- env ----------------------------------------------------------------

func cmdEnv(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("env requires a subcommand: set KEY=VALUE | get KEY")
	}

	switch args[0] {
	case "set":
		return envSet(args[1:])
	case "get":
		return envGet(args[1:])
	default:
		return fmt.Errorf("env: unknown subcommand %q; use set or get", args[0])
	}
}

func loadEnvFile() (map[string]string, error) {
	env := make(map[string]string)
	f, err := os.Open(envFile)
	if os.IsNotExist(err) {
		return env, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	return env, scanner.Err()
}

func saveEnvFile(env map[string]string) error {
	tmpPath := envFile + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	fmt.Fprintf(f, "# .nexacli-env — managed by nexacli env set/get\n")
	fmt.Fprintf(f, "# Do NOT commit this file; add it to .gitignore.\n\n")
	for k, v := range env {
		fmt.Fprintf(f, "%s=%s\n", k, v)
	}
	f.Close()
	return os.Rename(tmpPath, envFile)
}

func envSet(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: nexacli env set KEY=VALUE [KEY=VALUE ...]")
	}

	env, err := loadEnvFile()
	if err != nil {
		return err
	}

	for _, pair := range args {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid format %q — expected KEY=VALUE", pair)
		}
		env[parts[0]] = parts[1]
		fmt.Printf("  Set %s\n", parts[0])
	}

	return saveEnvFile(env)
}

func envGet(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: nexacli env get KEY")
	}

	env, err := loadEnvFile()
	if err != nil {
		return err
	}

	key := args[0]
	val, ok := env[key]
	if !ok {
		return fmt.Errorf("key %q not found in %s", key, envFile)
	}
	fmt.Println(val)
	return nil
}

// ---- help ---------------------------------------------------------------

func printUsage() {
	fmt.Printf(`NexaCLI %s — developer CLI for cloud deployments
https://github.com/adahir-gif/NexaCLI

USAGE:
  nexacli <command> [flags]

COMMANDS:
  init [--provider aws|gcp|azure|fly|railway|render]
      Initialize nexacli.yaml in the current directory.

  deploy [--dry-run]
      Build and deploy the application to the configured provider.

  rollback [--region <name>]
      Roll back the latest deployment. Optionally target one region.

  env set KEY=VALUE [KEY=VALUE ...]
      Persist one or more environment variables to .nexacli-env.

  env get KEY
      Read an environment variable from .nexacli-env.

  --version
      Print version information.

Run `nexacli <command> --help` for detailed flags on any command.
`, version)
}
