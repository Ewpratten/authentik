import { en, fr, tr } from "make-plural/plurals";

import { Messages, i18n } from "@lingui/core";
import { detect, fromNavigator, fromStorage, fromUrl } from "@lingui/detect-locale";
import { t } from "@lingui/macro";

import { messages as localeEN } from "../locales/en";
import { messages as localeFR_FR } from "../locales/fr_FR";
import { messages as localeDEBUG } from "../locales/pseudo-LOCALE";
import { messages as localeTR } from "../locales/tr";

export const LOCALES: {
    code: string;
    label: string;
    // eslint-disable-next-line @typescript-eslint/ban-types
    plurals: Function;
    locale: Messages;
}[] = [
    {
        code: "en",
        plurals: en,
        label: t`English`,
        locale: localeEN,
    },
    {
        code: "debug",
        plurals: en,
        label: t`Debug`,
        locale: localeDEBUG,
    },
    {
        code: "fr_FR",
        plurals: fr,
        label: t`French`,
        locale: localeFR_FR,
    },
    {
        code: "tr",
        plurals: tr,
        label: t`Turkish`,
        locale: localeTR,
    },
];

LOCALES.forEach((locale) => {
    i18n.loadLocaleData(locale.code, { plurals: locale.plurals });
    i18n.load(locale.code, locale.locale);
});

const DEFAULT_FALLBACK = () => "en";

export function autoDetectLanguage() {
    const detected =
        detect(fromUrl("lang"), fromStorage("lang"), fromNavigator(), DEFAULT_FALLBACK) ||
        DEFAULT_FALLBACK();
    if (detected in i18n._messages) {
        console.debug(`authentik/locale: Activating detected locale '${detected}'`);
        i18n.activate(detected);
    } else {
        console.debug(`authentik/locale: No locale for '${detected}', falling back to en`);
        i18n.activate(DEFAULT_FALLBACK());
    }
}
autoDetectLanguage();
