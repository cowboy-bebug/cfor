package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func costFilepath() string {
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(homeDir, ".local", "share")
	}
	return filepath.Join(dir, "cfor", "cost.json")
}

type Today string
type Cost float64
type Costs map[Today]Cost

func GetCosts() (Costs, error) {
	costFilePath := costFilepath()
	if costFilePath == "" {
		return nil, fmt.Errorf("could not determine cost file path")
	}

	if _, err := os.Stat(costFilePath); os.IsNotExist(err) {
		return nil, CostFileNotFoundError{}
	}

	costData, err := os.ReadFile(costFilePath)
	if err != nil {
		return nil, CostFileNotFoundError{}
	}

	var costs Costs
	if err := json.Unmarshal(costData, &costs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cost data: %w", err)
	}

	return costs, nil
}

func UpdateCost(cost float64) error {
	costFilePath := costFilepath()
	if costFilePath == "" {
		return fmt.Errorf("could not determine cost file path")
	}

	err := os.MkdirAll(filepath.Dir(costFilePath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	today := time.Now().Format("2006-01-02")
	costs := make(map[string]float64)
	costData, err := os.ReadFile(costFilePath)
	if err == nil && len(costData) > 0 {
		if err := json.Unmarshal(costData, &costs); err != nil {
			costs = make(map[string]float64)
		}
	}
	costs[today] += cost

	updatedData, err := json.MarshalIndent(costs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal costs: %w", err)
	}

	if err := os.WriteFile(costFilePath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write cost file: %w", err)
	}

	return nil
}
