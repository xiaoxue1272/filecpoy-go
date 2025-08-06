package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/sys/windows"
)

type Config struct {
	Src         string `json:"src"`
	Dest        string `json:"dest"`
	Extension   string `json:"extension"`
	IsOverided  bool   `json:"isOverided"`
	BufferSize  uint64 `json:"bufferSize"`
	OS          string `json:"os"`
}

var (
	config  Config
	scanner = bufio.NewScanner(os.Stdin)
	stdout  = color.New(color.FgGreen)
	stdwarn = color.New(color.FgYellow)
)

const (
	ASCIILogo = `
    ______ ____ __     ______ ______ ____   ____ __  __        ______ ____ 
   / ____//  _// /    / ____// ____// __ \ / __ \\ \/ /       / ____// __ \
  / /_    / / / /    / __/  / /    / / / // /_/ / \  /______ / / __ / / / /
 / __/  _/ / / /___ / /___ / /___ / /_/ // ____/  / //_____// /_/ // /_/ / 
/_/    /___//_____//_____/ \____/ \____//_/      /_/        \____/ \____/  
                                                                            
`
	Megabyte uint64 = 1024 * 1024
)

func main() {
	// 配置参数
	initConfig()

	// 创建目标根目录（如果不存在）
	if err := os.MkdirAll(config.Dest, 0755); err != nil {
		fmt.Printf("创建目标目录失败: %v\n", err)
		return
	}
	fileInfos, err := prepareSourceFiles()
	if err != nil {
		fmt.Printf("读取源目录失败: %v\n", err)
	}

	startProcessing(fileInfos)

	// 输出结果
	stdout.Println("\n=== 操作完成 ===")
	readScanText()
}

func initConfig() {
	stdout.Print(ASCIILogo)
	stdout.Println("请输入源文件夹")
	config.Src = readScanText()
	stdout.Println("请输入目标文件夹")
	config.Dest = readScanText()
	stdout.Println("请输入文件后缀名?")
	config.Extension = readScanText()
	stdout.Println("请输入拷贝文件字节缓冲区大小(MB/默认1MB)")
	bufferSize, err := strconv.ParseUint(readScanText(), 10, 64)
	if err != nil {
		bufferSize *= Megabyte
	}
	config.BufferSize = bufferSize
	stdout.Println("是否覆盖已存在文件? [y/N]")
	config.IsOverided = strings.ToLower(readScanText()) == "y"
	config.OS = strings.ToLower(runtime.GOOS)

	json, _ := json.MarshalIndent(&config, "\n", "\r\t")
	stdout.Printf("当前配置: \n%s\n", json)
}

func prepareSourceFiles() ([]os.FileInfo, error) {
	// 读取源目录
	entries, err := os.ReadDir(config.Src)
	if err != nil {
		return nil, err
	}

	// 转换成FileInfo
	return slices.Collect(func(yield func(os.FileInfo) bool) {
		for _, entry := range entries {
			if strings.HasSuffix(strings.ToLower(entry.Name()), strings.ToLower(config.Extension)) {
				info, err := entry.Info()
				if err != nil || !yield(info) {
					return
				}
			}
		}
	}), nil
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
		srcFilePath := filepath.Join(config.Src, filename)

		// 根据修改时间创建目标路径
		modTime := fileInfo.ModTime()
		dateFolder := modTime.Format("2006.01.02") // YYYY-MM-DD 格式
		destFolderPath := filepath.Join(config.Dest, dateFolder)
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
			if !config.IsOverided {
				stdwarn.Printf("文件已存在，跳过: %s\n", filename)
				skipped++
				continue
			}
		}

		if config.IsOverided {
			stdout.Printf("覆盖文件: %s → %s\n", filename, dateFolder)
		} else {
			stdout.Printf("复制文件: %s → %s\n", filename, dateFolder)
		}

		// 复制文件
		if err := copyFile(srcFilePath, destFilePath); err != nil {
			stdwarn.Printf("失败 [%s]: %v\n", filename, err)
			errors++
			continue
		}
		if config.OS == "windows" {
			if err := setWindowsFileAttributes(srcFilePath, destFilePath); err != nil {
				stdwarn.Printf("复制源文件属性失败 [%s]: %v\n", filename, err)
			}
		}
		if config.IsOverided {
			overrided++
		} else {
			processed++
		}
	}
	stdout.Printf("复制文件: %d\n跳过文件: %d\n覆盖文件: %d\n错误: %d\n", processed, skipped, overrided, errors)
}

func copyFile(src, dest string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.CopyBuffer(destination, source, make([]byte, config.BufferSize))
	return err
}

func setWindowsFileAttributes(src, dest string) error {
	// 转换路径（支持长路径）
	srcPtr, err := windows.UTF16PtrFromString(src)
	if err != nil {
		return err
	}
	dstPtr, err := windows.UTF16PtrFromString(dest)
	if err != nil {
		return err
	}

	// 打开源文件（带读权限和共享读）
	srcHandle, err := windows.CreateFile(
		srcPtr,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(srcHandle)

	// 获取源文件时间属性
	var creationTime, lastAccessTime, lastWriteTime windows.Filetime
	if err := windows.GetFileTime(srcHandle, &creationTime, &lastAccessTime, &lastWriteTime); err != nil {
		return err
	}

	// 打开目标文件
	destHandle, err := windows.CreateFile(
		dstPtr,
		windows.GENERIC_WRITE,
		0,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(destHandle)

	// 设置目标文件时间属性（必须在关闭文件前设置）
	if err := windows.SetFileTime(
		destHandle,
		&creationTime,
		&lastAccessTime,
		&lastWriteTime,
	); err != nil {
		return err
	}

	// 刷新缓存
	return windows.FlushFileBuffers(destHandle)
}

func readScanText() string {
	scanner.Scan()
	return scanner.Text()
}
