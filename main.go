package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/fatih/color"
)

type Config struct {
	src         string
	dest        string
	extension   string
	isOverided  bool
	isRecursive bool
}

var (
	config  = new(Config)
	stdout  = color.New(color.FgGreen)
	stdwarn = color.New(color.FgYellow)
)

func main() {
	// 配置参数
	stdout.Println(`
    ______ ____ __     ______ ______ ____   ____ __  __        ______ ____ 
   / ____//  _// /    / ____// ____// __ \ / __ \\ \/ /       / ____// __ \
  / /_    / / / /    / __/  / /    / / / // /_/ / \  /______ / / __ / / / /
 / __/  _/ / / /___ / /___ / /___ / /_/ // ____/  / //_____// /_/ // /_/ / 
/_/    /___//_____//_____/ \____/ \____//_/      /_/        \____/ \____/  
                                                                            
`)
	scanner := bufio.NewScanner(os.Stdin)
	stdout.Println("请输入源文件夹?")
	config.src = readScanText(scanner)
	stdout.Println("请输入目标文件夹?")
	config.dest = readScanText(scanner)
	stdout.Println("请输入文件后缀名?")
	config.extension = readScanText(scanner)
	stdout.Println("是否覆盖已存在文件? [y/N]")
	config.isOverided = strings.ToLower(readScanText(scanner)) == "y"
	stdout.Println("是否复制子文件夹及其内容? [y/N]")
	config.isRecursive = strings.ToLower(readScanText(scanner)) == "y"

	stdout.Println("=== 文件整理工具 ===")
	json, _ := json.Marshal(*config)
	stdout.Printf("当前配置: \n%s\n", json)

	// 创建目标根目录（如果不存在）
	if err := os.MkdirAll(config.dest, 0755); err != nil {
		fmt.Printf("创建目标目录失败: %v\n", err)
		return
	}

	// 读取源目录
	entries, err := os.ReadDir(config.src)
	if err != nil {
		fmt.Printf("读取源目录失败: %v\n", err)
		return
	}

	// 转换成FileInfo
	fileInfos := slices.Collect[os.FileInfo](func(yield func(os.FileInfo) bool) {
		for _, entry := range entries {
			if strings.HasSuffix(strings.ToLower(entry.Name()), strings.ToLower(config.extension)) {
				info, err := entry.Info()
				if err != nil || !yield(info) {
					return
				}
			}
		}
	})

	startProcessing(fileInfos)

	// 输出结果
	stdout.Println("\n=== 操作完成 ===")
	stdout.Println("\n按任意键退出程序...\n")
	readScanText(scanner)
}

func startProcessing(fileInfos []os.FileInfo) {
	stdout.Printf("开始复制文件, 数量: %d\n", len(fileInfos))
	// 处理文件
	processed, skipped, overrided, errors := 0, 0, 0, 0
	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			continue
		}

		filename := fileInfo.Name()
		srcFilePath := filepath.Join(config.src, filename)

		// 根据修改时间创建目标路径
		modTime := fileInfo.ModTime()
		dateFolder := modTime.Format("2006.01.02") // YYYY-MM-DD 格式
		destFolderPath := filepath.Join(config.dest, dateFolder)
		// 创建日期文件夹
		if err := os.MkdirAll(destFolderPath, 0755); err != nil {
			stdwarn.Printf("创建日期目录失败 [%s]: %v\n", dateFolder, err)
			errors++
			continue
		}
		destFilePath := filepath.Join(destFolderPath, filename)

		// 检查目标文件是否存在
		if _, err := os.Stat(destFilePath); err == nil {
			// 文件已存在
			if !config.isOverided {
				stdwarn.Printf("文件已存在，跳过: %s\n", filename)
				skipped++
				continue
			}
		}

		if config.isOverided {
			stdout.Printf("覆盖文件: %s → %s\n", filename, dateFolder)
		} else {
			stdout.Printf("复制文件: %s → %s\n", filename, dateFolder)
		}

		// 复制文件
		if err := copyFile(srcFilePath, destFilePath, mo); err != nil {
			stdwarn.Printf("失败 [%s]: %v\n", filename, err)
			errors++
		} else {
			if config.isOverided {
				overrided++
			} else {
				processed++
			}
		}
	}
	stdout.Printf("复制文件: %d\n跳过文件: %d\n覆盖文件: %d\n错误: %d\n", processed, skipped, overrided, errors)
}

func copyFile(src, dst string, modTime time.Time) error {
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
	os.Chtimes(dst, time.Now(), modTime)
	return err
}

func readScanText(scanner *bufio.Scanner) string {
	scanner.Scan()
	return scanner.Text()
}
