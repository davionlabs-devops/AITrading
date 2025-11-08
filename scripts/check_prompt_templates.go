package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// 检查数据库文件是否存在
	dbPath := "config.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Fatalf("数据库文件不存在: %s", dbPath)
	}

	// 打开数据库
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
	defer db.Close()

	// 查询所有交易员的提示词模板配置
	rows, err := db.Query(`
		SELECT 
			id, 
			user_id, 
			name, 
			COALESCE(system_prompt_template, 'default') as system_prompt_template,
			COALESCE(override_base_prompt, 0) as override_base_prompt,
			CASE 
				WHEN custom_prompt IS NULL OR custom_prompt = '' THEN 0 
				ELSE 1 
			END as has_custom_prompt
		FROM traders
		ORDER BY user_id, name
	`)
	if err != nil {
		log.Fatalf("查询失败: %v", err)
	}
	defer rows.Close()

	fmt.Println("=" + string(make([]byte, 80)))
	fmt.Println("交易员提示词模板配置检查")
	fmt.Println("=" + string(make([]byte, 80)))
	fmt.Printf("%-20s %-20s %-30s %-15s %-15s %-15s\n", 
		"交易员ID", "用户ID", "交易员名称", "模板", "覆盖默认", "有自定义提示词")
	fmt.Println("-" + string(make([]byte, 80)))

	var hasIssues bool
	for rows.Next() {
		var id, userID, name, template string
		var overrideBase, hasCustom int

		err := rows.Scan(&id, &userID, &name, &template, &overrideBase, &hasCustom)
		if err != nil {
			log.Printf("扫描行失败: %v", err)
			continue
		}

		// 检查是否有问题
		issue := ""
		if overrideBase == 1 && hasCustom == 1 {
			issue = "⚠️  覆盖默认+自定义提示词，模板将被忽略！"
			hasIssues = true
		} else if template == "default" || template == "" {
			issue = "ℹ️  使用默认模板"
		}

		fmt.Printf("%-20s %-20s %-30s %-15s %-15s %-15s %s\n",
			id[:min(20, len(id))],
			userID[:min(20, len(userID))],
			name[:min(30, len(name))],
			template,
			fmt.Sprintf("%v", overrideBase == 1),
			fmt.Sprintf("%v", hasCustom == 1),
			issue)
	}

	fmt.Println("-" + string(make([]byte, 80)))
	
	if hasIssues {
		fmt.Println("\n⚠️  发现问题：")
		fmt.Println("   如果 '覆盖默认' 为 true 且 '有自定义提示词' 为 true，")
		fmt.Println("   系统会完全忽略模板，只使用自定义提示词。")
		fmt.Println("\n建议：")
		fmt.Println("   1. 取消勾选 '覆盖默认提示词'，或")
		fmt.Println("   2. 清空 '附加提示词' 字段")
	} else {
		fmt.Println("\n✓ 所有配置正常")
	}

	// 检查可用的模板文件
	fmt.Println("\n" + "=" + string(make([]byte, 80)))
	fmt.Println("可用的提示词模板文件")
	fmt.Println("=" + string(make([]byte, 80)))
	
	templates := []string{"default", "adaptive", "adaptive_relaxed", "confidence_strategy", "Hansen", "nof1", "taro_long_prompts"}
	for _, tmpl := range templates {
		filePath := fmt.Sprintf("prompts/%s.txt", tmpl)
		if _, err := os.Stat(filePath); err == nil {
			fmt.Printf("✓ %s\n", tmpl)
		} else {
			fmt.Printf("✗ %s (文件不存在)\n", tmpl)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

