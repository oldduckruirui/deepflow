cmake_minimum_required(VERSION 3.0)
project(deepflow-ebpf-kernel)

# LLC ?= /usr/bin/llc
# CLANG ?= /usr/bin/clang
# LLVM_STRIP ?= /usr/bin/llvm-strip
# LLVM_OBJDUMP ?= /usr/bin/llvm-objdump
# CC ?= gcc
# TAEGET_KERN_SRC = socket_trace.c perf_profiler.c
# TAEGET_KERN_LL = $(TAEGET_KERN_SRC:c=ll)
# TAEGET_KERN_ELF = $(TAEGET_KERN_SRC:c=elf)
set(LCC "/usr/bin/llc" CACHE STRING "Path to llc")
set(CLANG "/usr/bin/clang" CACHE STRING "Path to clang")
set(LLVM_STRIP "/usr/bin/llvm-strip" CACHE STRING "Path to llvm-strip")
set(LLVM_OBJDUMP "/usr/bin/llvm-objdump" CACHE STRING "Path to llvm-objdump")
set(CC "/usr/bin/gcc" CACHE STRING "Path to gcc")
set(TAEGET_KERN_SRC "socket_trace.c" "perf_profiler.c" CACHE STRING "Source files")
set(TAEGET_KERN_LL "socket_trace.ll" "perf_profiler.ll" CACHE STRING "LL files")
set(TAEGET_KERN_ELF "socket_trace.elf" "perf_profiler.elf" CACHE STRING "ELF files")

execute_process(
  COMMAND sh -c "uname -m | sed 's/x86_64/x86/' | sed 's/aarch64/arm64/' | sed 's/ppc64le/powerpc/' | sed 's/mips.*/mips/' | tr -d '\n'"
  OUTPUT_VARIABLE ARCH
)

set(EBPF_C_FLAGS "-I${PROJECT_SOURCE_DIR}" "-I${PROJECT_SOURCE_DIR}/vmlinux" "-I${PROJECT_SOURCE_DIR}/include" "-D__BPF_TRACING__" "-DGROUP_LEADER_OFFSET_OVERRIDE=0"
"-DSTART_BOOTTIME_OFFSET_OVERRIDE=0" "-DSTART_BOOTTIME_VARNAME=real_start_time" CACHE STRING "EBPF C flags")
set(C_FLAGS "-std=gnu99" "-Wimplicit-function-declaration"
"-ffreestanding" "-fno-builtin" "-Wall"
"-Wno-deprecated-declarations"
"-Wno-gnu-variable-sized-type-not-at-end"
"-Wno-pragma-once-outside-header"
"-Wno-address-of-packed-member"
"-Wno-unknown-warning-option"
"-fno-color-diagnostics"
"-fno-unwind-tables"
"-fno-stack-protector"
"-fno-asynchronous-unwind-tables" "-g" "-O2")

file(GLOB_RECURSE KERNEL_SRC "*.h" "*.c")

function (build_ebpf_kernel_target target_name version extra_ebpf_c_flags)
  add_custom_command(
    OUTPUT ${PROJECT_SOURCE_DIR}/build/${target_name}_${version}.ll
    COMMAND ${CLANG} ${EBPF_C_FLAGS} ${extra_ebpf_c_flags} ${C_FLAGS} -emit-llvm -D__TARGET_ARCH_${ARCH} -o ${PROJECT_SOURCE_DIR}/build/${target_name}_${version}.ll -c ${PROJECT_SOURCE_DIR}/${target_name}.c
    DEPENDS ${KERNEL_SRC}
    COMMENT "Building ${target_name}_${version}.ll"
  )
  add_custom_command(
    OUTPUT ${PROJECT_SOURCE_DIR}/build/${target_name}_${version}.elf
    COMMAND ${LCC} -march=bpf -filetype=obj -mcpu=v2 -o ${PROJECT_SOURCE_DIR}/build/${target_name}_${version}.elf ${PROJECT_SOURCE_DIR}/build/${target_name}_${version}.ll
    DEPENDS ${PROJECT_SOURCE_DIR}/build/${target_name}_${version}.ll
    COMMENT "Building ${target_name}_${version}.elf"
  )
  add_custom_command(
    OUTPUT ${PROJECT_SOURCE_DIR}/build/${target_name}_${version}.objdump
    COMMAND ${LLVM_OBJDUMP} --source --debug-vars --line-numbers --symbol-description ${PROJECT_SOURCE_DIR}/build/${target_name}_${version}.elf > ${PROJECT_SOURCE_DIR}/build/${target_name}_${version}.objdump
    DEPENDS ${PROJECT_SOURCE_DIR}/build/${target_name}_${version}.elf
    COMMENT "Building ${target_name}_${version}.objdump"
  )
  add_custom_target(${target_name}_${version} ALL DEPENDS ${PROJECT_SOURCE_DIR}/build/${target_name}_${version}.objdump)
endfunction()


# ifeq ($(LINUX_VER_3_10_0),1)
# 	EXTRA_EBPF_CLAGS = -DLINUX_VER_3_10_0
# else ifeq ($(LINUX_VER_5_2_PLUS),1)
# 	EXTRA_EBPF_CLAGS = -DLINUX_VER_5_2_PLUS
# else ifeq ($(LINUX_VER_KYLIN),1)
# 	EXTRA_EBPF_CLAGS = -DLINUX_VER_KYLIN
# endif

build_ebpf_kernel_target(socket_trace common "")
build_ebpf_kernel_target(socket_trace 3_10_0 "-DLINUX_VER_3_10_0")
build_ebpf_kernel_target(socket_trace 5_2_plus "-DLINUX_VER_5_2_PLUS")
build_ebpf_kernel_target(socket_trace kylin "-DLINUX_VER_KYLIN")
build_ebpf_kernel_target(perf_profiler common "")