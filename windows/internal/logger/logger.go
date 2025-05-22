package logger

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// InitLog 初始化日志
func InitLog(logFilePath string, logMaxLines int) (*os.File, error) {
	// 创建日志目录
	dir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("无法创建日志目录: %v", err)
	}

	// 打开日志文件
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("无法创建日志文件: %v", err)
	}

	// 设置日志输出到文件和控制台
	multiWriter := io.MultiWriter(file, os.Stdout)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// 包装日志写入函数，限制行数
	log.SetOutput(&logWriter{
		file:     file,
		writer:   multiWriter,
		maxLines: logMaxLines,
	})

	return file, nil
}

// logWriter 自定义日志写入器，用于限制日志行数
type logWriter struct {
	file     *os.File
	writer   io.Writer
	maxLines int
}

// Write 实现 io.Writer 接口
func (w *logWriter) Write(p []byte) (n int, err error) {
	// 检查日志文件行数
	lines, err := countLines(w.file.Name())
	if err != nil {
		return 0, fmt.Errorf("统计日志行数失败: %v", err)
	}

	// 如果行数超过限制，删除最旧的一行
	if lines >= w.maxLines {
		if err := removeFirstLine(w.file.Name()); err != nil {
			return 0, fmt.Errorf("删除最旧日志行失败: %v", err)
		}
	}

	// 写入日志
	n, err = w.writer.Write(p)
	if err != nil {
		return n, err
	}

	return n, nil
}

// countLines 统计文件行数
func countLines(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("读取文件失败: %v", err)
	}

	return lines, nil
}

// removeFirstLine 删除文件的第一行
func removeFirstLine(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	// 读取文件内容，跳过第一行
	scanner := bufio.NewScanner(file)
	var lines []string
	scanner.Scan() // 跳过第一行
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取文件失败: %v", err)
	}

	// 重新写入文件
	if err := os.WriteFile(filePath, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	return nil
}
