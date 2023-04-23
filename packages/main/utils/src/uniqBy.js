export function uniqBy(array, iteratee) {
    return array.filter((item, index, self) => {
        return self.findIndex(i => iteratee(i) === iteratee(item)) === index;
    });
}
//# sourceMappingURL=uniqBy.js.map