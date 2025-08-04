package logger

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// InitLog 初始化日志并强制文件最大行数
func InitLog(logFilePath string, logMaxLines int) (*os.File, error) {
	// 确保日志目录存在
	dir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("无法创建日志目录: %v", err)
	}

	// 以追加模式打开或创建日志文件
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("无法打开日志文件: %v", err)
	}

	// 使用自定义写入器包装文件写入，超过行数时批量删除最旧日志行
	writer := &logWriter{
		path:       logFilePath,
		file:       file,
		maxLines:   logMaxLines,
		bufferSize: 50, // 每次批量删除的最旧日志行数
	}

	// 配置标准日志输出：写文件并安全写入控制台，忽略控制台写入错误
	log.SetOutput(io.MultiWriter(writer, &consoleWriter{os.Stdout}))
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	return file, nil
}

// consoleWriter 包装 os.Stdout，写入失败时忽略错误，避免服务环境下影响主写入
type consoleWriter struct {
	w io.Writer
}

func (cw *consoleWriter) Write(p []byte) (n int, err error) {
	n, _ = cw.w.Write(p) // 忽略错误
	return n, nil
}

// logWriter 用于批量删除最旧日志行，保持最大行数
type logWriter struct {
	path       string   // 日志文件路径
	file       *os.File // 当前文件句柄
	maxLines   int      // 最大行数限制
	bufferSize int      // 删除时一次跳过的行数
}

// Write 实现 io.Writer 接口：先写入，再在阈值外批量删除最旧日志行
func (w *logWriter) Write(p []byte) (n int, err error) {
	// 先写入文件
	n, err = w.file.Write(p)
	if err != nil {
		return n, err
	}

	// 统计当前行数
	lines, err := countLines(w.path)
	if err != nil {
		return n, nil // 统计失败则跳过
	}

	// 超过阈值时执行批量轮换
	if lines > w.maxLines+w.bufferSize {
		if err := w.rotate(); err != nil {
			return n, fmt.Errorf("日志文件轮换失败: %v", err)
		}
	}

	return n, nil
}

// rotate 原子化删除前 bufferSize 行并重新打开文件句柄
func (w *logWriter) rotate() error {
	// 打开原日志文件用于读取
	r, err := os.Open(w.path)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}
	defer r.Close()

	// 使用固定临时文件路径
	tmpPath := w.path + ".tmp"
	tmp, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer tmp.Close()

	// 跳过前 bufferSize 行并写入剩余内容
	scanner := bufio.NewScanner(r)
	skipped := 0
	for scanner.Scan() {
		if skipped < w.bufferSize {
			skipped++
			continue
		}
		if _, err := tmp.WriteString(scanner.Text() + "\n"); err != nil {
			return fmt.Errorf("写入临时文件失败: %v", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取日志文件失败: %v", err)
	}

	// 关闭并替换原文件
	w.file.Close()
	if err := os.Rename(tmpPath, w.path); err != nil {
		return fmt.Errorf("重命名临时文件失败: %v", err)
	}

	// 重新以追加模式打开文件
	file, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("重新打开日志文件失败: %v", err)
	}
	w.file = file
	return nil
}

// countLines 统计指定文件的行数
func countLines(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		count++
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("读取文件失败: %v", err)
	}

	return count, nil
}
