import * as yup from 'yup';
import { YupModel } from "../utils";

export const phoneValidation: YupModel<true, false> = {
    create: () => yup.object().shape({
    }),
    // Can't update an phone. Push notifications & other phone-related settings are updated elsewhere
}
