#!/bin/bash

RootPath=$(cd $(dirname ${BASH_SOURCE[0]})/..; pwd)
BuildPath=${RootPath}/build
BuildOutPath=${BuildPath}/out
BuildBinPath=${BuildPath}/bin
BuildDependsLibPath=${BuildPath}/lib
BuildDependsIncludePath=${BuildPath}/include
VendorPath=${RootPath}/vendor
DependsPath=${RootPath}/depends
use_clang=$(echo ${CC} | grep "clang" | grep -v "grep")
cgo_ldflags="-L${BuildDependsLibPath} -lrocksdb -lz -lbz2 -lsnappy -llz4 -lzstd -lstdc++"
if [ "${use_clang}" != "" ]; then
    cgo_ldflags="-L${BuildDependsLibPath} -lrocksdb -lz -lbz2 -lsnappy -llz4 -lzstd -lc++"
fi
cgo_cflags="-I${BuildDependsIncludePath}"
MODFLAGS=""
gomod=${2:-"on"}

if [ "${gomod}" == "off" ]; then
    MODFLAGS="-mod=vendor"
fi

if [ ! -d "${BuildOutPath}" ]; then
    mkdir ${BuildOutPath}
fi

if [ ! -d "${BuildBinPath}" ]; then
    mkdir ${BuildBinPath}
fi

if [ ! -d "${BuildDependsLibPath}" ]; then
    mkdir ${BuildDependsLibPath}
fi

if [ ! -d "${BuildDependsIncludePath}" ]; then
    mkdir ${BuildDependsIncludePath}
fi

RM=$(find /bin /sbin /usr/bin /usr/local -name "rm" | head -1)
if [[ "-x$RM" == "-x" ]] ; then
    RM=rm
fi

Version=$(git describe --abbrev=0 --tags 2>/dev/null)
BranchName=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)
CommitID=$(git rev-parse HEAD 2>/dev/null)
BuildTime=$(date +%Y-%m-%d\ %H:%M)
LDFlags="-X 'main.Version=${Version}' \
    -X 'main.CommitID=${CommitID}' \
    -X 'main.BranchName=${BranchName}' \
    -X 'main.BuildTime=${BuildTime}' \
    -w -s"

NPROC=$(nproc 2>/dev/null)
if [ -e /sys/fs/cgroup/cpu ] ; then
    NPROC=4
fi
NPROC=${NPROC:-"1"}

case $(uname -s | tr 'A-Z' 'a-z') in
    "linux"|"darwin")
        ;;
    *)
        echo "Current platform $(uname -s) not support";
        exit1;
        ;;
esac

CPUTYPE=${CPUTYPE} | tr 'A-Z' 'a-z'

build_zlib() {
    ZLIB_VER=1.2.13
    if [ -f "${BuildDependsLibPath}/libz.a" ]; then
        return 0
    fi

    if [ ! -d ${BuildOutPath}/zlib-${ZLIB_VER} ]; then
        tar -zxf ${DependsPath}/zlib-${ZLIB_VER}.tar.gz -C ${BuildOutPath}
    fi

    echo "build zlib..."
    pushd ${BuildOutPath}/zlib-${ZLIB_VER}
    CFLAGS='-fPIC' ./configure --static
    make
    if [ $? -ne 0 ]; then
        exit 1
    fi
    cp -f libz.a ${BuildDependsLibPath}
    cp -f zlib.h zconf.h ${BuildDependsIncludePath}
    popd
}

build_bzip2() {
    BZIP2_VER=1.0.6
    if [ -f "${BuildDependsLibPath}/libbz2.a" ]; then
        return 0
    fi

    if [ ! -d ${BuildOutPath}/bzip2-bzip2-${BZIP2_VER} ]; then
        tar -zxf ${DependsPath}/bzip2-bzip2-${BZIP2_VER}.tar.gz -C ${BuildOutPath}
        if [ "${use_clang}" != "" ]; then
            sed -i '18d' ${BuildOutPath}/bzip2-bzip2-${BZIP2_VER}/Makefile
        fi
    fi

    echo "build bzip2..."
    pushd ${BuildOutPath}/bzip2-bzip2-${BZIP2_VER}
    make CFLAGS='-fPIC -O2 -g -D_FILE_OFFSET_BITS=64'
    if [ $? -ne 0 ]; then
        exit 1
    fi
    cp -f libbz2.a ${BuildDependsLibPath}
    cp -f bzlib.h bzlib_private.h ${BuildDependsIncludePath}
    popd
}

build_lz4() {
    LZ4_VER=1.8.3
    if [ -f "${BuildDependsLibPath}/liblz4.a" ]; then
        return 0
    fi

    if [ ! -d ${BuildOutPath}/lz4-${LZ4_VER} ]; then
        tar -zxf ${DependsPath}/lz4-${LZ4_VER}.tar.gz -C ${BuildOutPath}
    fi

    echo "build lz4..."
    pushd ${BuildOutPath}/lz4-${LZ4_VER}/lib
    make CFLAGS='-fPIC -O2'
    if [ $? -ne 0 ]; then
        exit 1
    fi
    cp -f liblz4.a ${BuildDependsLibPath}
    cp -f lz4frame_static.h lz4.h lz4hc.h lz4frame.h ${BuildDependsIncludePath}
    popd
}

build_zstd() {
    ZSTD_VER=1.4.0
    if [ -f "${BuildDependsLibPath}/libzstd.a" ]; then
        return 0
    fi

    if [ ! -d ${BuildOutPath}/zstd-${ZSTD_VER} ]; then
        tar -zxf ${DependsPath}/zstd-${ZSTD_VER}.tar.gz -C ${BuildOutPath}
    fi

    echo "build zstd..."
    pushd ${BuildOutPath}/zstd-${ZSTD_VER}/lib
    make CFLAGS='-fPIC -O2'
    if [ $? -ne 0 ]; then
        exit 1
    fi
    cp -f libzstd.a ${BuildDependsLibPath}
    cp -f zstd.h common/zstd_errors.h deprecated/zbuff.h dictBuilder/zdict.h ${BuildDependsIncludePath}
    popd
}


build_snappy() {
    SNAPPY_VER=1.1.7
    if [ -f "${BuildDependsLibPath}/libsnappy.a" ]; then
        return 0
    fi

    if [ ! -d ${BuildOutPath}/snappy-${SNAPPY_VER} ]; then
        tar -zxf ${DependsPath}/snappy-${SNAPPY_VER}.tar.gz -C ${BuildOutPath}
    fi

    echo "build snappy..."
    mkdir ${BuildOutPath}/snappy-${SNAPPY_VER}/build
    pushd ${BuildOutPath}/snappy-${SNAPPY_VER}/build
    cmake -DCMAKE_POSITION_INDEPENDENT_CODE=ON -DSNAPPY_BUILD_TESTS=OFF .. && make
    if [ $? -ne 0 ]; then
        exit 1
    fi
    cp -f libsnappy.a ${BuildDependsLibPath}
    cp -f ../snappy-c.h ../snappy-sinksource.h ../snappy.h snappy-stubs-public.h ${BuildDependsIncludePath}
    popd
}

build_rocksdb() {
    ROCKSDB_VER=6.3.6
    if [ -f "${BuildDependsLibPath}/librocksdb.a" ]; then
        return 0
    fi

    if [ ! -d ${BuildOutPath}/rocksdb-${ROCKSDB_VER} ]; then
        tar -zxf ${DependsPath}/rocksdb-${ROCKSDB_VER}.tar.gz -C ${BuildOutPath}
        pushd ${BuildOutPath} > /dev/null
        sed -i '1069s/newf/\&newf/' rocksdb-${ROCKSDB_VER}/db/db_impl/db_impl_compaction_flush.cc
        sed -i '1161s/newf/\&newf/' rocksdb-${ROCKSDB_VER}/db/db_impl/db_impl_compaction_flush.cc
        sed -i '412s/pair/\&pair/' rocksdb-${ROCKSDB_VER}/options/options_parser.cc
        sed -i '63s/std::mutex/mutable std::mutex/' rocksdb-${ROCKSDB_VER}/util/channel.h
        popd
    fi

    echo "build rocksdb..."
    pushd ${BuildOutPath}/rocksdb-${ROCKSDB_VER}
    if [ "${use_clang}" != "" ]; then
        FLAGS="-Wno-error=deprecated-copy -Wno-error=pessimizing-move -Wno-error=shadow -Wno-error=unused-but-set-variable"
    else
        CCMAJOR=`gcc -dumpversion | awk -F. '{print $1}'`
        if [ ${CCMAJOR} -ge 9 ]; then
            FLAGS="-Wno-error=deprecated-copy -Wno-error=pessimizing-move"
        fi
    fi
    FLAGS="${FLAGS} -Wno-unused-variable -Wno-unused-function"
    PORTABLE=1 make EXTRA_CXXFLAGS="-fPIC ${FLAGS} -DZLIB -DBZIP2 -DSNAPPY -DLZ4 -DZSTD -I${BuildDependsIncludePath}" static_lib
    if [ $? -ne 0 ]; then
        exit 1
    fi
    make install-static INSTALL_PATH=${BuildPath}
    strip -S -x ${BuildDependsLibPath}/librocksdb.a
    popd
}

init_gopath() {
    export GO111MODULE=${gomod}
    export GOPATH=$HOME/tmp/kfasthdfs/go

    mkdir -p $GOPATH/src/github.com/JohnMicl/kubefasthdfs
    SrcPath=$GOPATH/src/github.com/JohnMicl/kubefasthdfs/kubefasthdfs
    if [ -L "$SrcPath" ]; then
        $RM -f $SrcPath
    fi
    if [  ! -e "$SrcPath" ] ; then
        ln -s $RootPath $SrcPath 2>/dev/null
    fi
}

pre_build() {
    build_zlib
    build_bzip2
    build_lz4
    build_zstd
    build_snappy
    build_rocksdb

    export CGO_CFLAGS=${cgo_cflags}
    export CGO_LDFLAGS="${cgo_ldflags}"
    echo $CGO_CFLAGS
    echo $CGO_LDFLAGS

    init_gopath
}

build_server() {
    pushd $SrcPath >/dev/null
    echo -n "build kubefasthdfs-server   "
    CGO_ENABLED=1 go build ${MODFLAGS} -gcflags=all=-trimpath=${SrcPath} -asmflags=all=-trimpath=${SrcPath} -ldflags="${LDFlags}" -o ${BuildBinPath}/kubefasthdfs-server ${SrcPath}/cmd/*.go && echo "success" || echo "failed"
    popd >/dev/null
}

clean() {
    $RM -rf ${BuildBinPath}
}

cmd=${1:-"all"}

if [ "${cmd}" == "dist_clean" ]; then
    dist_clean
    exit 0
elif [ "${cmd}" == "clean" ]; then
    clean
    exit 0
fi

pre_build

case "$cmd" in
    "all")
        build_server
        ;;
    "server")
        build_server
        ;;
    *)
        ;;
esac