package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// 配置参数
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("请输入源文件夹")
	sourceDir := readScanText(scanner)
	fmt.Println("请输入目标文件夹")
	targetDir := readScanText(scanner)
	fmt.Println("请输入文件后缀名")
	fileExt := readScanText(scanner)
	fmt.Println("是否覆盖已存在文件 [y/N]")
	overwriteExisting := strings.ToLower(readScanText(scanner)) == "y"

	fmt.Println("=== 文件整理工具 ===")
	fmt.Printf("源目录: %s\n目标目录: %s\n文件类型: %s\n",
		sourceDir, targetDir, fileExt)

	// 创建目标根目录（如果不存在）
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		fmt.Printf("创建目标目录失败: %v\n", err)
		return
	}

	// 读取源目录
	files, err := os.ReadDir(sourceDir)
	if err != nil {
		fmt.Printf("读取源目录失败: %v\n", err)
		return
	}

	// 处理文件
	processed, skipped, overrided, errors := 0, 0, 0, 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		// 检查文件后缀
		if !strings.HasSuffix(strings.ToLower(filename), strings.ToLower(fileExt)) {
			continue
		}

		sourcePath := filepath.Join(sourceDir, filename)

		// 获取文件信息
		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			fmt.Printf("获取文件信息失败 [%s]: %v\n", filename, err)
			errors++
			continue
		}

		// 根据修改时间创建目标路径
		modTime := fileInfo.ModTime()
		dateFolder := modTime.Format("2006.01.02") // YYYY-MM-DD 格式
		targetPath := filepath.Join(targetDir, dateFolder)
		// 创建日期文件夹
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			fmt.Printf("创建日期目录失败 [%s]: %v\n", dateFolder, err)
			errors++
			continue
		}
		targetPath = filepath.Join(targetPath, filename)

		// 检查目标文件是否存在
		if _, err := os.Stat(targetPath); err == nil {
			// 文件已存在
			if !overwriteExisting {
				fmt.Printf("文件已存在，跳过: %s\n", filename)
				skipped++
				continue
			}
		}

		if overwriteExisting {
			fmt.Printf("覆盖文件: %s → %s\n", filename, dateFolder)
		} else {
			fmt.Printf("复制文件: %s → %s\n", filename, dateFolder)
		}

		// 复制文件
		if err := copyFile(sourcePath, targetPath); err != nil {
			fmt.Printf("失败 [%s]: %v\n", filename, err)
			errors++
		} else {
			if overwriteExisting {
				overrided++
			} else {
				processed++
			}
		}
	}

	// 输出结果
	fmt.Println("\n=== 操作完成 ===")
	fmt.Printf("处理文件: %d\n跳过文件: %d\n覆盖文件: %d\n错误: %d\n", processed, skipped, overrided, errors)

	fmt.Println("\n按任意键退出程序...")
	readScanText(scanner)
}

// 高效复制文件函数
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.CopyBuffer(destination, source, make([]byte, 1024*1024))
	return err
}

func readScanText(scanner *bufio.Scanner) string {
	scanner.Scan()
	return scanner.Text()
}
