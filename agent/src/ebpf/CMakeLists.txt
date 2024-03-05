cmake_minimum_required(VERSION 3.0)
project(deepflow-ebpf)

set(ARCH ${CMAKE_SYSTEM_PROCESSOR})


execute_process(
  COMMAND mkdir -p ${PROJECT_SOURCE_DIR}/kernel/build
)
add_subdirectory(kernel)

# CLANG ?= /usr/bin/clang
# CC ?= gcc
# GNU_CC ?= gcc
# MUSL_CC ?= musl-gcc
# OBJDIR ?= .
# STATIC_OBJDIR := $(OBJDIR)/staticobjs

set(CLANG "/usr/bin/clang" CACHE FILEPATH "Path to Clang compiler")
set(CC "/usr/bin/gcc" CACHE STRING "C Compiler")
set(GNU_CC "/usr/bin/gcc" CACHE STRING "GNU C Compiler")
set(MUSL_CC "/usr/bin/musl-gcc" CACHE STRING "Musl C Compiler")
set(OBJDIR "." CACHE PATH "Object directory")
set(STATIC_OBJDIR "${OBJDIR}/staticobjs" CACHE PATH "Static object directory")

if(CC MATCHES "musl")
  set(IS_MUSL 1)
else()
  set(IS_MUSL 0)
endif()

# check gcc, musl and clang version
set(GCC_MIN_VERSION "8.0.0")
set(MUSL_MIN_VERSION "10.0.0")
set(CLANG_MIN_VERSION "10.0.0")

execute_process(
  COMMAND ${CC} --version
  OUTPUT_VARIABLE GCC_VERSION_OUTPUT
)
string(REGEX MATCH "[0-9]+\\.[0-9]+\\.[0-9]+" GCC_VERSION ${GCC_VERSION_OUTPUT})
if (GCC_VERSION)
  message(STATUS "GCC Version: ${GCC_VERSION}")
  if (GCC_VERSION VERSION_LESS ${GCC_MIN_VERSION})
    message(FATAL_ERROR "GCC version must be at least ${GCC_MIN_VERSION}")
  endif()
else()
  message(WARNING "Failed to get GCC version")
endif()

execute_process(
  COMMAND ${MUSL_CC} --version
  OUTPUT_VARIABLE MUSL_VERSION_OUTPUT
)
string(REGEX MATCH "[0-9]+\\.[0-9]+\\.[0-9]+" MUSL_VERSION ${MUSL_VERSION_OUTPUT})
if (MUSL_VERSION)
  message(STATUS "Musl Version: ${MUSL_VERSION}")
  if (MUSL_VERSION VERSION_LESS ${MUSL_MIN_VERSION})
    message(FATAL_ERROR "Musl version must be at least ${MUSL_MIN_VERSION}")
  endif()
else()
  message(WARNING "Failed to get Musl version")
endif()

execute_process(
  COMMAND ${CLANG} --version
  OUTPUT_VARIABLE CLANG_VERSION_OUTPUT
)
string(REGEX MATCH "[0-9]+\\.[0-9]+\\.[0-9]+" CLANG_VERSION ${CLANG_VERSION_OUTPUT})
if (CLANG_VERSION)
  message(STATUS "Clang Version: ${CLANG_VERSION}")
  if (CLANG_VERSION VERSION_LESS ${CLANG_MIN_VERSION})
    message(FATAL_ERROR "Clang version must be at least ${CLANG_MIN_VERSION}")
  endif()
else()
  message(WARNING "Failed to get Clang version")
endif()

# makefile variables
# JAVA_TOOL := deepflow-jattach
# JAVA_AGENT_VERSION := 1
# JAVA_AGENT_GNU_SO := df_java_agent_v$(JAVA_AGENT_VERSION).so
# JAVA_AGENT_MUSL_SO := df_java_agent_musl_v$(JAVA_AGENT_VERSION).so
# JAVA_AGENT_SO := $(JAVA_AGENT_GNU_SO) $(JAVA_AGENT_MUSL_SO)
# JAVA_AGENT_SRC := user/profile/java/agent.c
# JAVA_AGENT_MACROS := -DAGENT_LIB_NAME="\"$(JAVA_AGENT_GNU_SO)\"" -DAGENT_MUSL_LIB_NAME="\"$(JAVA_AGENT_MUSL_SO)\""
set(JAVA_TOOL ${PROJECT_SOURCE_DIR}/deepflow-jattach)
set(JAVA_AGENT_VERSION "1")
set(JAVA_AGENT_GNU_SRC ${PROJECT_SOURCE_DIR}/user/profile/java_agent_so_gnu.c)
set(JAVA_AGENT_MUSL_SRC ${PROJECT_SOURCE_DIR}/user/profile/java_agent_so_musl.c)
set(JATTACH_SRC ${PROJECT_SOURCE_DIR}//user/profile/deepflow_jattach_bin.c)
set(JAVA_AGENT_GNU_SO ${PROJECT_SOURCE_DIR}/df_java_agent_v${JAVA_AGENT_VERSION}.so)
set(JAVA_AGENT_GNU_SO_WITHOUT_PATH "df_java_agent_v${JAVA_AGENT_VERSION}.so")
set(JAVA_AGENT_MUSL_SO ${PROJECT_SOURCE_DIR}/df_java_agent_musl_v${JAVA_AGENT_VERSION}.so)
set(JAVA_AGENT_MUSL_SO_WITHOUT_PATH "df_java_agent_musl_v${JAVA_AGENT_VERSION}.so")
set(JAVA_AGENT_SO ${JAVA_AGENT_GNU_SO} ${JAVA_AGENT_MUSL_SO})
set(JAVA_AGENT_SRC ${PROJECT_SOURCE_DIR}/user/profile/java/agent.c)
set(JAVA_AGENT_MACROS "-DAGENT_LIB_NAME=\"\\\"${JAVA_AGENT_GNU_SO_WITHOUT_PATH}\\\"\"" "-DAGENT_MUSL_LIB_NAME=\"\\\"${JAVA_AGENT_MUSL_SO_WITHOUT_PATH}\\\"\"")



add_executable(bintobuffer tools/bintobuffer.c)
set_target_properties(bintobuffer PROPERTIES
RUNTIME_OUTPUT_DIRECTORY "${PROJECT_SOURCE_DIR}/tools"
)

set(C_FLAGS "-std=gnu99" "-g" "-O2" "-ffunction-sections" "-fdata-sections" "-fPIC" "-Wall" "-Wno-strict-aliasing" "-Wno-sign-compare" "-Wno-unused-parameter"
"-Wno-missing-field-initializers" "-fno-omit-frame-pointer" "-I/usr/lib/jvm/java/include" "-I/usr/lib/jvm/java/include/linux" "-I${PROJECT_SOURCE_DIR}"
"-Wformat-truncation=0" ${JAVA_AGENT_MACROS})

add_custom_command(
  OUTPUT ${JAVA_AGENT_GNU_SO} ${JAVA_AGENT_GNU_SRC}
  COMMAND ${GNU_CC} ${C_FLAGS} -shared -o ${JAVA_AGENT_GNU_SO} ${JAVA_AGENT_SRC}
  COMMAND ${PROJECT_SOURCE_DIR}/tools/bintobuffer ${JAVA_AGENT_GNU_SO} ${JAVA_AGENT_GNU_SRC} java_agent_so_gnu
  DEPENDS ${JAVA_AGENT_SRC} bintobuffer
  COMMENT "Building GNU Java Agent"
)
add_custom_command(
  OUTPUT ${JAVA_AGENT_MUSL_SO} ${JAVA_AGENT_MUSL_SRC}
  COMMAND ${MUSL_CC} ${C_FLAGS} -shared -o ${JAVA_AGENT_MUSL_SO} ${JAVA_AGENT_SRC}
  COMMAND ${PROJECT_SOURCE_DIR}/tools/bintobuffer ${JAVA_AGENT_MUSL_SO} ${JAVA_AGENT_MUSL_SRC} java_agent_so_musl
  DEPENDS ${JAVA_AGENT_SRC} bintobuffer
  COMMENT "Building Musl Java Agent"
)
add_custom_command(
  OUTPUT ${JATTACH_SRC} ${JAVA_TOOL}
  COMMAND ${GNU_CC} ${C_FLAGS} -DJAVA_AGENT_ATTACH_TOOL ${PROJECT_SOURCE_DIR}/user/profile/java/df_jattach.c ${PROJECT_SOURCE_DIR}/user/log.c ${PROJECT_SOURCE_DIR}/user/common.c ${PROJECT_SOURCE_DIR}/libs/jattach/build/libjattach.a -o ${JAVA_TOOL} -ldl -lpthread
  COMMAND rm -rf ${JATTACH_SRC}
  COMMAND ${PROJECT_SOURCE_DIR}/tools/bintobuffer ${JAVA_TOOL} ${JATTACH_SRC} deepflow_jattach_bin
  DEPENDS ${JAVA_AGENT_GNU_SO} ${JAVA_AGENT_MUSL_SO} ${PROJECT_SOURCE_DIR}/user/profile/java/df_jattach.c ${PROJECT_SOURCE_DIR}/user/log.c ${PROJECT_SOURCE_DIR}/user/common.c ${PROJECT_SOURCE_DIR}/libs/jattach/build/libjattach.a bintobuffer
  COMMENT "Touching Java Agent"
)
add_custom_command(
  OUTPUT ${PROJECT_SOURCE_DIR}/libs/jattach/build/libjattach.a
  COMMAND patch -p1 -d ${PROJECT_SOURCE_DIR}/libs/jattach -Ns --no-backup-if-mismatch < ${PROJECT_SOURCE_DIR}/libs/0001-Add-compilation-for-static-lib.patch || true
  COMMAND make -sC ${PROJECT_SOURCE_DIR}/libs/jattach dll --no-print-directory
)

function(compile_socket_trace_elf version flags)
  add_custom_command(
    OUTPUT ${PROJECT_SOURCE_DIR}/user/socket_trace_bpf_${version}.c
    COMMAND ${PROJECT_SOURCE_DIR}/tools/bintobuffer ${PROJECT_SOURCE_DIR}/kernel/build/socket_trace_${version}.elf ${PROJECT_SOURCE_DIR}/user/socket_trace_bpf_${version}.c socket_trace_${version}_ebpf_data
    DEPENDS bintobuffer socket_trace_${version}
    COMMENT "Building socket_trace.elf for kernel version ${version}"
  )
endfunction()

function(compile_perf_profiler_elf version flag)
  add_custom_command(
    OUTPUT ${PROJECT_SOURCE_DIR}/user/perf_profiler_bpf_${version}.c
    COMMAND ${PROJECT_SOURCE_DIR}/tools/bintobuffer ${PROJECT_SOURCE_DIR}/kernel/build/perf_profiler_${version}.elf ${PROJECT_SOURCE_DIR}/user/perf_profiler_bpf_${version}.c perf_profiler_${version}_ebpf_data
    DEPENDS bintobuffer ${JATTACH_SRC} perf_profiler_${version}
    COMMENT "Building perf_profiler.elf for kernel version ${version}"
  )
endfunction()

set(SOCKET_TRACE_BPF_SRC ${PROJECT_SOURCE_DIR}/user/socket_trace_bpf_common.c ${PROJECT_SOURCE_DIR}/user/socket_trace_bpf_3_10_0.c ${PROJECT_SOURCE_DIR}/user/socket_trace_bpf_5_2_plus.c ${PROJECT_SOURCE_DIR}/user/socket_trace_bpf_kylin.c)
compile_socket_trace_elf(common "")
compile_socket_trace_elf(3_10_0 LINUX_VER_3_10_0=1)
compile_socket_trace_elf(5_2_plus LINUX_VER_5_2_PLUS=1)
compile_socket_trace_elf(kylin LINUX_VER_KYLIN=1)

set(PERF_PROFILER_BPF_SRC ${PROJECT_SOURCE_DIR}/user/perf_profiler_bpf_common.c)
compile_perf_profiler_elf(common "")

# extract all source files
file(GLOB_RECURSE LIBTRACE_SRC "user/*.h" "user/*.c")
# remove bihash_template.h and bihash_template.c
list(REMOVE_ITEM LIBTRACE_SRC ${PROJECT_SOURCE_DIR}/user/bihash_template.h)
list(REMOVE_ITEM LIBTRACE_SRC ${PROJECT_SOURCE_DIR}/user/bihash_template.c)
# remove ctrl_tracer.c
list(REMOVE_ITEM LIBTRACE_SRC ${PROJECT_SOURCE_DIR}/user/ctrl_tracer.c)
# append java agent source files
list(APPEND LIBTRACE_SRC ${JAVA_AGENT_GNU_SRC})
list(APPEND LIBTRACE_SRC ${JAVA_AGENT_MUSL_SRC})
# append socket trace bpf source files
list(APPEND LIBTRACE_SRC ${SOCKET_TRACE_BPF_SRC} ${PERF_PROFILER_BPF_SRC})

set(CMAKE_C_COMPILER ${CC})
add_library(trace STATIC ${LIBTRACE_SRC})
target_compile_options(trace PRIVATE ${C_FLAGS})
set_target_properties(trace PROPERTIES
  POSITION_INDEPENDENT_CODE ON
  ARCHIVE_OUTPUT_DIRECTORY ${PROJECT_SOURCE_DIR}
)

add_custom_command(
  OUTPUT ${PROJECT_SOURCE_DIR}/deepflow-ebpfctl
  COMMAND ${CC} ${C_FLAGS} --static -g -O2 ${PROJECT_SOURCE_DIR}/user/ctrl_tracer.c ${PROJECT_SOURCE_DIR}/user/ctrl.c ${PROJECT_SOURCE_DIR}/libtrace.a -o ${PROJECT_SOURCE_DIR}/deepflow-ebpfctl -lelf -lz -lpthread
  DEPENDS ${PROJECT_SOURCE_DIR}/user/ctrl_tracer.c ${PROJECT_SOURCE_DIR}/user/ctrl.c trace
  COMMENT "Building deepflow-ebpfctl"
)

add_custom_target(
  tools
  DEPENDS ${PROJECT_SOURCE_DIR}/deepflow-ebpfctl
)

add_custom_target(
  rust-sample
  DEPENDS ${PROJECT_SOURCE_DIR}/socket-tracer ${PROJECT_SOURCE_DIR}/profiler
)

file(GLOB_RECURSE RUST_SAMPLE_SRC "${PROJECT_SOURCE_DIR}/samples/rust/socket-tracer/src/*")
list(REMOVE_ITEM RUST_SAMPLE_SRC ${PROJECT_SOURCE_DIR}/samples/rust/socket-tracer/src/ebpf.rs)


if(${ARCH} MATCHES "aarch64")
  add_custom_command(
    OUTPUT ${PROJECT_SOURCE_DIR}/socket-tracer
    COMMAND rm -rf ${PROJECT_SOURCE_DIR}/samples/rust/socket-tracer/src/ebpf.rs
    COMMAND cp ${PROJECT_SOURCE_DIR}/mod.rs ${PROJECT_SOURCE_DIR}/samples/rust/socket-tracer/src/ebpf.rs
    COMMAND rm -rf ${PROJECT_SOURCE_DIR}/samples/rust/socket-tracer/target
    COMMAND bash -c \"if [ ${IS_MUSL} -eq 1 ]; then \\\n
              cd ${PROJECT_SOURCE_DIR}/samples/rust/socket-tracer/ && cargo build --release --target aarch64-unknown-linux-musl\; \\\n
            else \\\n
              cd ${PROJECT_SOURCE_DIR}/samples/rust/socket-tracer/ && cargo build --release\; \\\n
            fi\"
    DEPENDS ${RUST_SAMPLE_SRC} ${PROJECT_SOURCE_DIR}/mod.rs ${JATTACH_SRC} trace
  )
elseif(${ARCH} MATCHES "x86_64")
  add_custom_command(
    OUTPUT ${PROJECT_SOURCE_DIR}/socket-tracer
    COMMAND rm -rf ${PROJECT_SOURCE_DIR}/samples/rust/socket-tracer/src/ebpf.rs
    COMMAND cp ${PROJECT_SOURCE_DIR}/mod.rs ${PROJECT_SOURCE_DIR}/samples/rust/socket-tracer/src/ebpf.rs
    COMMAND rm -rf ${PROJECT_SOURCE_DIR}/samples/rust/socket-tracer/target
    COMMAND bash -c \" if [ ${IS_MUSL} -eq 1 ]; then \\\n
              cd ${PROJECT_SOURCE_DIR}/samples/rust/socket-tracer/ && RUSTFLAGS='-D warnings -C force-frame-pointers=yes' cargo build --release --target x86_64-unknown-linux-musl\; \\\n
            else \\\n
              cd ${PROJECT_SOURCE_DIR}/samples/rust/socket-tracer/ && RUSTFLAGS=-Cforce-frame-pointers=yes cargo build --release\; \\\n
            fi \"
    DEPENDS ${RUST_SAMPLE_SRC} ${PROJECT_SOURCE_DIR}/mod.rs ${JATTACH_SRC} trace
  )
else()
  message(FATAL_ERROR "Unsupported architecture: ${ARCH}")
endif()

if(${ARCH} MATCHES "aarch64")
  add_custom_command(
    OUTPUT ${PROJECT_SOURCE_DIR}/profiler
    COMMAND rm -rf ${PROJECT_SOURCE_DIR}/samples/rust/profiler/src/ebpf.rs
    COMMAND cp ${PROJECT_SOURCE_DIR}/mod.rs ${PROJECT_SOURCE_DIR}/samples/rust/profiler/src/ebpf.rs
    COMMAND rm -rf ${PROJECT_SOURCE_DIR}/samples/rust/profiler/target
    COMMAND bash -c \"if [ ${IS_MUSL} -eq 1 ]; then \\\n
              cd ${PROJECT_SOURCE_DIR}/samples/rust/profiler/ && cargo build --release --target aarch64-unknown-linux-musl\; \\\n
            else \\\n
              cd ${PROJECT_SOURCE_DIR}/samples/rust/profiler/ && cargo build --release\; \\\n
            fi\"
    DEPENDS ${RUST_SAMPLE_SRC} ${PROJECT_SOURCE_DIR}/mod.rs ${JAVA_TOOL} trace
  )
elseif(${ARCH} MATCHES "x86_64")
  add_custom_command(
    OUTPUT ${PROJECT_SOURCE_DIR}/profiler
    COMMAND rm -rf ${PROJECT_SOURCE_DIR}/samples/rust/profiler/src/ebpf.rs
    COMMAND cp ${PROJECT_SOURCE_DIR}/mod.rs ${PROJECT_SOURCE_DIR}/samples/rust/profiler/src/ebpf.rs
    COMMAND rm -rf ${PROJECT_SOURCE_DIR}/samples/rust/profiler/target
    COMMAND bash -c \" if [ ${IS_MUSL} -eq 1 ]; then \\\n
              cd ${PROJECT_SOURCE_DIR}/samples/rust/profiler/ && cargo build --release --target x86_64-unknown-linux-musl\; \\\n
            else \\\n
              cd ${PROJECT_SOURCE_DIR}/samples/rust/profiler/ && RUSTFLAGS=-Cforce-frame-pointers=yes cargo build --release\; \\\n
            fi \"
    DEPENDS ${RUST_SAMPLE_SRC} ${PROJECT_SOURCE_DIR}/mod.rs ${JAVA_TOOL} trace
  )
else()
  message(FATAL_ERROR "Unsupported architecture: ${ARCH}")
endif()

set_property(DIRECTORY PROPERTY ADDITIONAL_MAKE_CLEAN_FILES
  "${PROJECT_SOURCE_DIR}/deepflow-ebpfctl" 
  "${PROJECT_SOURCE_DIR}/deepflow-jattach"
)