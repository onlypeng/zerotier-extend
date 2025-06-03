@echo off
chcp 936 >nul
REM ZerotierExtend 服务控制脚本
REM 交互式菜单，用户选择操作
set EXE_PATH=%~dp0update_planet.exe

:menu
cls
echo ==============================
echo ZerotierExtend 服务控制菜单
echo ==============================
echo 1. 安装服务
echo 2. 卸载服务
echo 3. 启动服务
echo 4. 停止服务
echo 5. 重启服务
echo 6. 退出
echo ==============================
set /p choice=请选择操作 (1-6): 

if "%choice%"=="1" goto install
if "%choice%"=="2" goto uninstall
if "%choice%"=="3" goto start
if "%choice%"=="4" goto stop
if "%choice%"=="5" goto restart
if "%choice%"=="6" exit /b 0

echo 无效的选择，请输入 1 到 6 之间的数字。
pause
goto menu

:install
%EXE_PATH% install
pause
goto menu

:uninstall
%EXE_PATH% uninstall
pause
goto menu

:start
%EXE_PATH% start
pause
goto menu

:stop
%EXE_PATH% stop
pause
goto menu

:restart
%EXE_PATH% restart
pause
goto menu