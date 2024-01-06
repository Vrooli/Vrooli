const reactI18nextMock = {
    useTranslation: () => ({
        t: (str: string) => str,
        i18n: {
            changeLanguage: jest.fn(),
        },
    }),
    withTranslation: () => (Component: any) => {
        Component.defaultProps = { ...Component.defaultProps, t: (str: string) => str };
        return Component;
    },
};

export default reactI18nextMock;
